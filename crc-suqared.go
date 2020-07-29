package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chanzuckerberg/crc-squared/crcsquared"
	"github.com/jessevdk/go-flags"
	"golang.org/x/exp/mmap"
)

type options struct {
	PartSize    int64 `short:"p" long:"part-size" description:"Part size in bytes" default:"1024"`
	Concurrency int   `short:"c" long:"concurrency" description:"Concurrency"`
	Mmap        bool  `short:"m" long:"mmap" description:"Use mmap for downloads"`
	Positional  struct {
		Filepath string `description:"file path to checksum"`
	} `positional-args:"yes"`
}

// mainWork is a functional version of main that does all of the actual computation of main but can be easily tested
func mainWork(args []string) (uint32, error) {
	var opts options
	_, err := flags.ParseArgs(&opts, args)
	// go-flags will handle printing so we just exit with 0 for the help command and 2 for other parsing errors
	if err != nil {
		if strings.HasPrefix(err.Error(), "Usage") {
			os.Exit(0)
		}
		os.Exit(2)
	}
	checksumOpts := crcsquared.ParallelChecksumOptions{
		PartSize:    opts.PartSize,
		Concurrency: opts.Concurrency,
	}

	stats, err := os.Stat(opts.Positional.Filepath)
	if err != nil {
		return 0, err
	}
	length := stats.Size()

	var readerAt io.ReaderAt
	if opts.Mmap {
		readerAt, err = mmap.Open(opts.Positional.Filepath)
	} else {
		readerAt, err = os.Open(opts.Positional.Filepath)
	}
	if err != nil {
		return 0, err
	}

	return crcsquared.ParallelCRC32CChecksum(&readerAt, length, checksumOpts)
}

func main() {
	checksum, err := mainWork(os.Args[1:])
	if err != nil {
		panic(err)
	}
	fmt.Println(checksum)
}
