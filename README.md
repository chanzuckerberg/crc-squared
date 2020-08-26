# crc-squared [![Latest Version](https://img.shields.io/github/release/chanzuckerberg/crc-squared.svg?style=flat?maxAge=86400)](https://github.com/chanzuckerberg/crc-squared/releases) ![Check](https://github.com/chanzuckerberg/crc-squared/workflows/Check/badge.svg) [![codecov](https://codecov.io/gh/chanzuckerberg/crc-squared/branch/master/graph/badge.svg)](https://codecov.io/gh/chanzuckerberg/crc-squared) [![GitHub license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://github.com/chanzuckerberg/idseq-web/blob/master/LICENSE) ![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)



crc-squared is a CLI and library for computing crc-32c checksums fast. The performance is achieved by:

- Parallelism
- [The intel crc32 instruction](https://www.sciencedirect.com/science/article/abs/pii/S002001901100319X)
- mmap (optional)

## Work in Progress

This repo is currently in its infancy. I currently wouldn't recommend you use it. Here is what I feel it needs before it's ready:

- **Benchmarking**
- **Improving the parallelization**: The current algorithm reaps a lot of the benefits of parallelism but there are some bottlenecks I could remove
- **Documentation**
- **Removing the length requirement**: Currently you need to know the length of what you want to checksum in advance. Ideally I would rework this so you could use just `io.ReaderAt`.

## Installation

### Linux

#### Debian (Ubuntu/Mint)

Download and install the `.deb`:

```bash
RELEASES=chanzuckerberg/crc-squared/releases
VERSION=$(curl https://api.github.com/repos/${RELEASES}/latest | jq -r .name | sed s/^v//)
DOWNLOAD=crc-squared_${VERSION}_linux_amd64.deb
curl -L https://github.com/${RELEASES}/download/v${VERSION}/${DOWNLOAD} -o crc-squared.deb
sudo dpkg -i crc-squared.deb
rm crc-squared.deb
```

#### Fedora (RHEL/CentOS)

Download and install the `.rpm`:

```bash
RELEASES=chanzuckerberg/crc-squared/releases
VERSION=$(curl https://api.github.com/repos/${RELEASES}/latest | jq -r .name | sed s/^v//)
DOWNLOAD=crc-squared_${VERSION}_linux_amd64.rpm
curl -L https://github.com/${RELEASES}/download/v${VERSION}/${DOWNLOAD} -o crc-squared.rpm
sudo rpm -i crc-squared.rpm
rm crc-squared.rpm
```

### MacOS

Install via homebrew:

```bash
brew tap chanzuckerberg/tap
brew install crc-squared
```

### Binary

Download the appropriate binary for your platform:

```bash
RELEASES=chanzuckerberg/crc-squared/releases
PLATFORM=#linux,darwin,windows
VERSION=$(curl https://api.github.com/repos/${RELEASES}/latest | jq -r .name | sed s/^v//)
DOWNLOAD=crc-squared_${VERSION}_${PLATFORM}_amd64.tar.gz
curl -L https://github.com/${RELEASES}/download/v${VERSION}/${DOWNLOAD} | tar zx
```

### Windows

Follow instructions for binary, with `windows`.

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
