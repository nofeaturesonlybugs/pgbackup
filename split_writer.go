package pgbackup

import (
	"fmt"
	"os"
)

// SplitWriter splits data written to it into sequentially numbered files.
type SplitWriter struct {
	// Base path and filename to which the generated suffix will be added.
	Basepath string
	// The size in bytes that each file should be.
	SplitSize int
	// The file suffix length.
	SuffixLength int
	//
	fileNo    int
	remaining int
	dfd       *os.File
}

// Close closes the writer.
func (w *SplitWriter) Close() error {
	if w.dfd != nil {
		w.remaining = 0
		return w.dfd.Close()
	}
	return nil
}

// Write writes a chunk of data to the splitter.
func (w *SplitWriter) Write(p []byte) (int, error) {
	var wrote, total int
	var err error
	//
	for size := len(p); size > 0; size = size - wrote {
		//
		// Current dest file may be full.
		if w.remaining == 0 && w.dfd != nil {
			if err = w.dfd.Close(); err != nil {
				return total, err
			}
			w.dfd = nil
		}
		//
		// If w.dfd is nil we need to open our dest file.
		if w.dfd == nil {
			dname := w.Basepath + fmt.Sprintf(".%0[1]*d", w.SuffixLength, w.fileNo)
			if w.dfd, err = os.Create(dname); err != nil {
				return total, err
			}
			w.fileNo++
			w.remaining = w.SplitSize
		}
		//
		if size > w.remaining {
			wrote, err = w.dfd.Write(p[0:w.remaining])
			p = p[w.remaining:]
		} else {
			wrote, err = w.dfd.Write(p)
		}
		total = total + wrote
		w.remaining = w.remaining - wrote
		if err != nil {
			return total, err
		}
	}
	//
	return total, nil
}
