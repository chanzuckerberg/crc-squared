package crcsquared

import (
	"errors"
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

// CRC32CChecksum computes the crc32c checksum of some data
func CRC32CChecksum(data []byte) uint32 {
	return crc32.Checksum(data, crc32q)
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
		checksums <- partChecksum{
			Start:    partRange.Start,
			End:      partRange.End,
			Checksum: CRC32CChecksum(data),
			Error:    err,
		}
	}
}

// ParallelChecksumOptions are the options for running a parallelized checksum
type ParallelChecksumOptions struct {
	Concurrency int
	PartSize    int64
}

type partChecksumbufferNode struct {
	Self partChecksum
	Next *partChecksumbufferNode
}

// partChecksumBuffer holds a linked list of part checksums
// If part checksums are only inserted via AddFuse they are guaranteed
// to be ordered by the end of each part and be maximally fused (all
// adjacent part checksums are fused).
type partChecksumBuffer struct {
	Head *partChecksumbufferNode
}

// AddFuse adds a part checksum to the linked list in order, ordered by the end of each part.
// Then, starting from the part checksum prior to the new part checksum (or itself in case the new
// part checksum is inserted at the head) fuses any adjacent checksums until the next checksum
// is not adjacent. This guarantees that the linked list is in order and maximally fused provided
// the list is only updated via this function.
func (buff *partChecksumBuffer) AddFuse(p partChecksum) {
	current := buff.Head
	if buff.Head == nil || buff.Head.Self.End > p.End {
		buff.Head = &partChecksumbufferNode{
			Self: p,
			Next: buff.Head,
		}
		current = buff.Head
	} else {
		for current.Next != nil && p.End > current.Next.Self.End {
			current = current.Next
		}
		current.Next = &partChecksumbufferNode{
			Self: p,
			Next: current.Next,
		}
	}

	for current.Next != nil && current.Self.End == current.Next.Self.Start {
		current.Self.Checksum = crc32combine.CRC32Combine(crc32.Castagnoli, current.Self.Checksum, current.Next.Self.Checksum, current.Next.Self.End-current.Next.Self.Start)
		current.Self.End = current.Next.Self.End
		current.Next = current.Next.Next
	}
}

// FinalChecksum returns the checksum of the head of the buffer if the head exists and is the only element, otherwise it returns an error
func (buff *partChecksumBuffer) FinalChecksum() (uint32, error) {
	if buff.Head == nil {
		return 0, errors.New("no partial checksums added to buffer")
	}
	if buff.Head.Next != nil {
		return 0, errors.New("unfused partial checksums still in buffer")
	}
	return buff.Head.Self.Checksum, nil
}

// ParallelCRC32CChecksum computes the crc32c checksum for a readerAt using parallelism
func ParallelCRC32CChecksum(readerAt io.ReaderAt, length int64, opts ParallelChecksumOptions) (uint32, error) {
	concurrency := opts.Concurrency
	if concurrency == 0 {
		concurrency = runtime.NumCPU()
	}

	numParts := length / opts.PartSize
	if length%opts.PartSize > 0 {
		numParts++
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

	// The big idea behind this algorithm is to do as much fusion as possible as soon as possible
	// As long as the algorithm is computing checksums the time to fuse checksums is free. Things
	// only slow down when we fuse checksums after we are done computing all of them. To cut down
	// the number of fusions performed after checksums this algorithm performs as many fusions as
	// possible after each checksum is computed. The checksums are computed roughly in order so
	// adjacent checksums will likely finish close together. This spreads out the fusion work in
	// the average case.
	var buffer partChecksumBuffer
	for i := int64(0); i < numParts; i++ {
		buffer.AddFuse(<-partChecksums)
	}

	return buffer.FinalChecksum()
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
