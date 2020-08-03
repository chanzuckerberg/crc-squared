package crcsquared

import (
	"hash/crc32"
	"io"
	"os"
	"runtime"

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
	Start int64
	End   int64
}

type partChecksum struct {
	Start    int64
	End      int64
	Checksum uint32
	Error    error
}

func checksumWorker(readerAt *io.ReaderAt, partRanges <-chan partRange, checksums chan<- partChecksum) {
	for partRange := range partRanges {
		data := make([]byte, partRange.End-partRange.Start)
		_, err := (*readerAt).ReadAt(data, partRange.Start)
		if err != nil {
			checksums <- partChecksum{
				Start:    partRange.Start,
				End:      partRange.End,
				Checksum: 0,
				Error:    err,
			}
		} else {
			checksum, err := CRC32CChecksum(data)
			checksums <- partChecksum{
				Start:    partRange.Start,
				End:      partRange.End,
				Checksum: checksum,
				Error:    err,
			}
		}
	}
}

// ParallelChecksumOptions are the options for running a parallelized checksum
type ParallelChecksumOptions struct {
	Concurrency int
	PartSize    int64
}

type bufferNode struct {
	Self partChecksum
	Next *bufferNode
}

// ParallelCRC32CChecksum computes the crc32c checksum for a readerAt using parallelism
func ParallelCRC32CChecksum(readerAt io.ReaderAt, length int64, opts ParallelChecksumOptions) (uint32, error) {
	concurrency := opts.Concurrency
	if concurrency == 0 {
		concurrency = runtime.NumCPU()
	}

	numParts := length / opts.PartSize
	lastPartSize := length % opts.PartSize
	if lastPartSize > 0 {
		numParts++
	} else {
		lastPartSize = opts.PartSize
	}

	partRanges := make(chan partRange, numParts)
	partChecksums := make(chan partChecksum, numParts)

	for w := 0; w < concurrency; w++ {
		go checksumWorker(&readerAt, partRanges, partChecksums)
	}

	for i := int64(0); i < numParts-1; i++ {
		partRanges <- partRange{
			Start: i * opts.PartSize,
			End:   (i + 1) * opts.PartSize,
		}
	}

	partRanges <- partRange{
		Start: (numParts - 1) * opts.PartSize,
		End:   length,
	}

	close(partRanges)

	checksum := uint32(0)
	end := int64(0)
	var buffer *bufferNode = nil

	for i := int64(0); i < numParts; i++ {
		p := <-partChecksums
		if p.Error != nil {
			return 0, p.Error
		}

		if p.Start == 0 {
			checksum = p.Checksum
			end = p.End
			continue
		}

		if buffer == nil || p.Start < buffer.Self.Start {
			buffer = &bufferNode{
				Self: p,
				Next: buffer,
			}
		} else {
			current := buffer
			for current.Next != nil && p.Start > current.Next.Self.Start {
				current = current.Next
			}
			current.Next = &bufferNode{
				Self: p,
				Next: current.Next,
			}

		}

		for buffer != nil && end == buffer.Self.Start {
			checksum = crc32combine.CRC32Combine(crc32.Castagnoli, checksum, buffer.Self.Checksum, buffer.Self.End-buffer.Self.Start)
			end = buffer.Self.End
			buffer = buffer.Next
		}
	}

	for buffer != nil && end == buffer.Self.Start {
		checksum = crc32combine.CRC32Combine(crc32.Castagnoli, checksum, buffer.Self.Checksum, buffer.Self.End-buffer.Self.Start)
		end = buffer.Self.End
		buffer = buffer.Next
	}

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
