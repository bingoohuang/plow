package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type LogFile struct {
	File *os.File
	Pos  int64
	sync.Mutex
}

func (f *LogFile) WriteString(s string) {
	f.Lock()
	f.File.WriteString(s)
	f.Unlock()
}

func (f *LogFile) Write(b *bytes.Buffer) {
	f.Lock()
	b.WriteTo(f.File)
	f.Unlock()
}

func (f *LogFile) MarkPos() {
	f.Lock()
	f.Pos, _ = f.File.Seek(0, io.SeekCurrent)
	f.Unlock()
}

func CreateLogFile(verbose int) *LogFile {
	if verbose < 2 {
		return nil
	}

	f, err := os.CreateTemp(".", "blow_"+time.Now().Format(`20060102150405`)+"_"+"*.log")
	ExitIfErr(err)

	fmt.Printf("Log details to: %s\n", f.Name())
	return &LogFile{
		File: f,
	}
}

func (f *LogFile) GetLastLog(tag string) string {
	f.Lock()
	defer f.Unlock()

	data, _ := ReadFileFromPos(f.File, f.Pos)
	return string(data)
}

func (f *LogFile) Close() error {
	f.Lock()
	defer f.Unlock()

	return f.File.Close()
}

func ReadFileFromPos(f *os.File, pos int64) ([]byte, error) {
	var size int64
	if info, err := f.Stat(); err == nil {
		size = info.Size()
	}
	size++ // one byte for final read at EOF

	_, err := f.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, err
	}

	size -= pos

	// If a file claims a small size, read at least 512 bytes.
	// In particular, files in Linux's /proc claim size 0 but
	// then do not work right if read in small pieces,
	// so an initial read of 1 byte would not work correctly.
	if size < 512 {
		size = 512
	}

	data := make([]byte, 0, size)
	for {
		if len(data) >= cap(data) {
			d := append(data[:cap(data)], 0)
			data = d[:len(data)]
		}
		n, err := f.Read(data[len(data):cap(data)])
		data = data[:len(data)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return data, err
		}
	}
}

func getLastLog(f *os.File, tag string) string {
	found := false
	ch := make([]byte, len(tag))
	var cursor int64
	for {
		cursor--
		_, err := f.Seek(cursor, io.SeekEnd)
		if err != nil {
			return ""
		}

		n, err := f.Read(ch)
		if err != nil {
			return ""
		}

		if n == len(tag) && string(ch) == tag { // stop if we find last log
			found = true
			break
		}
	}

	if !found {
		return ""
	}

	buffer := make([]byte, -cursor)
	n, _ := f.Read(buffer)
	return "\n" + tag + string(buffer[:n])
}
