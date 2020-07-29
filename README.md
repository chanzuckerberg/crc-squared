# crc-squared

[![codecov](https://codecov.io/gh/chanzuckerberg/crc-squared/branch/master/graph/badge.svg)](https://codecov.io/gh/chanzuckerberg/crc-squared)

crc-squared is a CLI and library for computing crc-32c checksums fast. The performance is achieved by:

- Parallelism
- [The intel crc32 instruction](https://www.sciencedirect.com/science/article/abs/pii/S002001901100319X)
- mmap (optional)

## Work in Progress

This repo is currently in its infancy. I currently wouldn't recommend you use it. Here is what I feel it needs before it's ready:

- **Benchmarking**
- **Improving the parallelization**: The current algorithm reaps a lot of the benefits of parallelism but there are some bottlenecks I could remove
- **Documentation**
- **Release Process**
- **Removing the length requirement**: Currently you need to know the length of what you want to checksum in advance. Ideally I would rework this so you could use just `io.ReaderAt`.

## Installation

To install clone the repo and build with `go build`.

## Usage

```
Usage:
  crc-squared [OPTIONS] [Filepath]

Application Options:
  -p, --part-size=   Part size in bytes (default: 1024)
  -c, --concurrency= Concurrency
  -m, --mmap         Use mmap for downloads

Help Options:
  -h, --help         Show this help message

Arguments:
  Filepath:          file path to checksum
```