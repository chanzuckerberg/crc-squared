package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/chanzuckerberg/crc-squared/crcsquared"
	"github.com/jessevdk/go-flags"
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
	checksumFileOpts := crcsquared.ParallelChecksumFileOptions{
		PartSize:    opts.PartSize,
		Concurrency: opts.Concurrency,
		Mmap:        opts.Mmap,
	}
	return crcsquared.ParallelCRC32CChecksumFile(opts.Positional.Filepath, checksumFileOpts)
}

func main() {
	checksum, err := mainWork(os.Args[1:])
	if err != nil {
		panic(err)
	}
	fmt.Println(checksum)
}
