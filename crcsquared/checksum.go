package crcsquared

import (
	"hash/crc32"
	"io"
	"os"
	"sync"

	"github.com/vimeo/go-util/crc32combine"
	"golang.org/x/exp/mmap"
)

// This ensures that we use the crc32c system command if available
//   I stepped though the code to verify
var crc32q *crc32.Table = crc32.MakeTable(crc32.Castagnoli)

// CRC32CChecksum computes the crc32c checksum of a file
func CRC32CChecksum(data []byte) (uint32, error) {
	return crc32.Checksum(data, crc32q), nil
}

type partRange struct {
	Chunk int64
	Start int64
	End   int64
}

type partChecksum struct {
	Chunk    int64
	Checksum uint32
	Error    error
}

func checksumWorker(readerAt *io.ReaderAt, partRanges <-chan partRange, checksums chan<- partChecksum) {
	for partRange := range partRanges {
		data := make([]byte, partRange.End-partRange.Start)
		_, err := (*readerAt).ReadAt(data, partRange.Start)
		if err != nil {
			checksums <- partChecksum{
				Chunk:    partRange.Chunk,
				Checksum: 0,
				Error:    err,
			}
		} else {
			checksum, err := CRC32CChecksum(data)
			checksums <- partChecksum{
				Chunk:    partRange.Chunk,
				Checksum: checksum,
				Error:    err,
			}
		}
	}
}

func parallelCRCFuse(checksums *[]uint32, numParts, partSize, length, lastPartSize int64) uint32 {
	nextPower := numParts << 1
	for n := int64(1); n < nextPower; n <<= 1 {
		var wg sync.WaitGroup

		for i := int64(0); i+n < numParts; i += 2 * n {
			wg.Add(1)
			go func(i int64) {
				len2 := partSize * n
				prevLen := (i + n) * partSize
				if len2+prevLen > length {
					len2 = length - prevLen
				} else if i+n == numParts-n {
					len2 -= (partSize - lastPartSize)
				}
				(*checksums)[i] = crc32combine.CRC32Combine(crc32.Castagnoli, (*checksums)[i], (*checksums)[i+n], len2)
				wg.Done()
			}(i)
		}
		wg.Wait()
	}
	return (*checksums)[0]
}

// ParallelChecksumOptions are the options for running a parallelized checksum
type ParallelChecksumOptions struct {
	Concurrency int
	PartSize    int64
}

// ParallelCRC32CChecksum computes the crc32c checksum for a readerAt using parallelism
func ParallelCRC32CChecksum(readerAt io.ReaderAt, length int64, opts ParallelChecksumOptions) (uint32, error) {
	numParts := length / opts.PartSize
	lastPartSize := length % opts.PartSize
	if lastPartSize > 0 {
		numParts++
	} else {
		lastPartSize = opts.PartSize
	}

	partRanges := make(chan partRange, numParts)
	partChecksums := make(chan partChecksum, numParts)
	checksums := make([]uint32, numParts)

	for w := 0; w < opts.Concurrency; w++ {
		go checksumWorker(&readerAt, partRanges, partChecksums)
	}

	for i := int64(0); i < numParts-1; i++ {
		partRanges <- partRange{
			Chunk: i,
			Start: i * opts.PartSize,
			End:   (i + 1) * opts.PartSize,
		}
	}

	partRanges <- partRange{
		Chunk: numParts - 1,
		Start: (numParts - 1) * opts.PartSize,
		End:   length,
	}

	close(partRanges)

	for i := int64(0); i < numParts; i++ {
		partChecksum := <-partChecksums
		if partChecksum.Error != nil {
			return 0, partChecksum.Error
		}
		checksums[partChecksum.Chunk] = partChecksum.Checksum
	}

	checksum := parallelCRCFuse(&checksums, numParts, opts.PartSize, length, lastPartSize)

	return checksum, nil
}

// ParallelChecksumFileOptions are the options for running a parallelized checksum on a file
type ParallelChecksumFileOptions struct {
	Concurrency int
	PartSize    int64
	Mmap        bool
}

type readAtCloser interface {
	io.ReaderAt
	Close() error
}

// ParallelCRC32CChecksumFile is a convenience function that opens a file and computes the crc32c checksum with ParallelCRC32CChecksum
func ParallelCRC32CChecksumFile(filepath string, opts ParallelChecksumFileOptions) (uint32, error) {
	// This also ensures we don't crash with a segfault when opening a non-existent file with mmap
	stat, err := os.Stat(filepath)
	if err != nil {
		return 0, err
	}

	var f readAtCloser
	if opts.Mmap {
		f, err = mmap.Open(filepath)
	} else {
		f, err = os.Open(filepath)
	}
	defer f.Close()
	if err != nil {
		return 0, err
	}

	parallelChecksumOptions := ParallelChecksumOptions{
		Concurrency: opts.Concurrency,
		PartSize:    opts.PartSize,
	}

	return ParallelCRC32CChecksum(f, stat.Size(), parallelChecksumOptions)
}
