package crcsquared

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
)

func dummyBytes(n int64, seed int64) []byte {
	bytes := make([]byte, n)
	if seed > 0 {
		rand.Seed(seed)
	}
	rand.Read(bytes)
	return bytes
}

type dummyReaderAt struct {
	data []byte
}

func (d dummyReaderAt) ReadAt(p []byte, off int64) (int, error) {
	n := 0
	for i := off; i < int64(len(p))+off && i < int64(len(d.data)); i++ {
		n++
		p[i-off] = d.data[i]
	}
	return n, nil
}

func (d dummyReaderAt) Size() int64 {
	return int64(len(d.data))
}

func newDummyReaderAt(n int64, seed int64) dummyReaderAt {
	return dummyReaderAt{data: dummyBytes(n, seed)}
}

func TestCRC32CChecksum(t *testing.T) {
	str := "sample bytes"
	bytes := []byte(str)
	expectedChecksum := uint32(1168601409)
	checksum, err := CRC32CChecksum(bytes)
	if err != nil {
		t.Errorf("Expected error to be nil; got %s", err)
	}

	if checksum != expectedChecksum {
		t.Errorf("Expected CRC32CChecksum(\"%s\") to equal %d; got %d", str, expectedChecksum, checksum)
	}
	fmt.Println(checksum)
}

func TestParallelCRC32CChecksum(t *testing.T) {
	for concurrency := 0; concurrency < 20; concurrency += 3 {
		for partsize := int64(1); partsize < 2000; partsize *= 10 {
			for length := int64(1); length < 5000; length *= 10 {
				readerAt := newDummyReaderAt(length, 42)
				expectedChecksum, err := CRC32CChecksum([]byte(readerAt.data))
				if err != nil {
					t.Errorf("Computing in-memory checksum for comparison errored with: %s", err)
					t.FailNow()
				}

				actualChecksum, err := ParallelCRC32CChecksum(readerAt, readerAt.Size(), ParallelChecksumOptions{
					Concurrency: concurrency,
					PartSize:    partsize,
				})

				if err != nil {
					t.Errorf("ParallelCRC32CChecksum errored with %s", err)
					t.FailNow()
				}

				if actualChecksum != expectedChecksum {
					t.Errorf("Expected parallel CRC32C Checksum to Equal %d %d", actualChecksum, expectedChecksum)
				}
			}
		}
	}
}

func TestParallelCRC32CChecksumFile(t *testing.T) {
	tmp, err := ioutil.TempFile("/tmp", "crc-squared-")
	if err != nil {
		t.Errorf("Creating temporary file for parallel checksum errored with %s", err)
		t.FailNow()
	}
	defer os.Remove(tmp.Name())

	bytes := dummyBytes(5000, 88)
	n, err := tmp.Write(bytes)
	if n != len(bytes) {
		t.Errorf("Didn't write all sample bytes to file wanted %d, got %d", len(bytes), n)
		t.FailNow()
	}
	if err != nil {
		t.Errorf("Writing sample bytes to file errored with: %s", err)
		t.FailNow()
	}

	expectedChecksum, err := CRC32CChecksum(bytes)
	if err != nil {
		t.Errorf("Computing in-memory checksum for comparison errored with: %s", err)
		t.FailNow()
	}

	actualChecksum, err := ParallelCRC32CChecksumFile(tmp.Name(), ParallelChecksumFileOptions{
		Concurrency: 10,
		PartSize:    10,
	})
	if err != nil {
		t.Errorf("ParallelCRC32CChecksum errored with %s", err)
		t.FailNow()
	}

	if actualChecksum != expectedChecksum {
		t.Errorf("Expected parallel CRC32C Checksum to Equal %d %d", actualChecksum, expectedChecksum)
	}
}

func TestParallelCRC32CChecksumFileMmap(t *testing.T) {
	tmp, err := ioutil.TempFile("/tmp", "crc-squared-")
	if err != nil {
		t.Errorf("Creating temporary file for parallel checksum errored with %s", err)
		t.FailNow()
	}
	defer os.Remove(tmp.Name())

	bytes := dummyBytes(5000, 88)
	n, err := tmp.Write(bytes)
	if n != len(bytes) {
		t.Errorf("Didn't write all sample bytes to file wanted %d, got %d", len(bytes), n)
		t.FailNow()
	}
	if err != nil {
		t.Errorf("Writing sample bytes to file errored with: %s", err)
		t.FailNow()
	}

	expectedChecksum, err := CRC32CChecksum(bytes)
	if err != nil {
		t.Errorf("Computing in-memory checksum for comparison errored with: %s", err)
		t.FailNow()
	}

	actualChecksum, err := ParallelCRC32CChecksumFile(tmp.Name(), ParallelChecksumFileOptions{
		Concurrency: 10,
		PartSize:    10,
		Mmap:        true,
	})
	if err != nil {
		t.Errorf("ParallelCRC32CChecksum errored with %s", err)
		t.FailNow()
	}

	if actualChecksum != expectedChecksum {
		t.Errorf("Expected parallel CRC32C Checksum to Equal %d %d", actualChecksum, expectedChecksum)
	}
}

func TestParallelCRC32CChecksumFileNonExistent(t *testing.T) {
	filename := string(dummyBytes(10, 9))
	expectedMessage := fmt.Sprintf("stat %s: no such file or directory", filename)
	_, err := ParallelCRC32CChecksumFile(filename, ParallelChecksumFileOptions{
		Concurrency: 10,
		PartSize:    10,
	})
	if err == nil {
		t.Errorf("Expected ParallelCRC32CChecksumFile to error on non-existent file but it did not")
		t.FailNow()
	}
	if err.Error() != expectedMessage {
		t.Errorf("Expected ParallelCRC32CChecksumFile on non-existent file to error with message \"%s\" but the error message was \"%s\"", expectedMessage, err.Error())
	}
}

func Benchmark(t *testing.B) {
	tmp, err := ioutil.TempFile("/tmp", "crc-squared-")
	if err != nil {
		t.Errorf("Creating temporary file for parallel checksum errored with %s", err)
		t.FailNow()
	}
	defer os.Remove(tmp.Name())

	bytes := dummyBytes(5000, 88)
	n, err := tmp.Write(bytes)
	if n != len(bytes) {
		t.Errorf("Didn't write all sample bytes to file wanted %d, got %d", len(bytes), n)
		t.FailNow()
	}
	if err != nil {
		t.Errorf("Writing sample bytes to file errored with: %s", err)
		t.FailNow()
	}

	expectedChecksum, err := CRC32CChecksum(bytes)
	if err != nil {
		t.Errorf("Computing in-memory checksum for comparison errored with: %s", err)
		t.FailNow()
	}

	actualChecksum, err := ParallelCRC32CChecksumFile(tmp.Name(), ParallelChecksumFileOptions{
		Concurrency: 10,
		PartSize:    10,
	})
	t.FailNow()
	if err != nil {
		t.Errorf("ParallelCRC32CChecksum errored with %s", err)
		t.FailNow()
	}

	if actualChecksum != expectedChecksum {
		t.Errorf("Expected parallel CRC32C Checksum to Equal %d %d", actualChecksum, expectedChecksum)
	}
}
