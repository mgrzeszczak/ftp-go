package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

type fdata struct {
	frames   uint32
	filename string
}

func packFileData(f *fdata) []byte {
	name := []byte(f.filename)
	out := make([]byte, 4+len(name))
	binary.BigEndian.PutUint32(out[0:4], f.frames)
	copy(out[4:], name)
	return out
}

func unpack(bytes []byte) *fdata {
	out := fdata{}
	out.frames = binary.BigEndian.Uint32(bytes[:4])
	out.filename = string(bytes[4:])
	return &out
}

func startFileWriter(fc <-chan *frame, firstFrame *frame) {
	var filedata *fdata
	var file *os.File
	var frameCount uint32

	filedata = unpack(firstFrame.content)

	fname := filedata.filename
	i := 0
	for {
		if _, err := os.Stat(fname); err == nil {
			i++
			fname = fmt.Sprintf("%s%v", filedata.filename, i)
		} else {
			break
		}
	}

	openedFile, err := os.Create(fname)
	if err != nil {
		// TODO: handle gracefully
		panic(fmt.Sprintf("Failed to open file %s\n", filedata.filename))
	}

	file = openedFile
	defer func() {
		log.Printf("File received %v\n", filedata.filename)
		file.Close()
	}()

	log.Printf("Began receiving file %v\n", filedata.filename)

	var progress float32

	for {
		f := <-fc
		frameCount++
		//log.Printf("Received frame %v. Content:\n\n%v\n\n", frameCount, f.content)
		var wrote uint32
		for wrote < f.len {
			n, err := file.Write(f.content[wrote:])
			if err != nil {
				// TODO: handle gracefully
				panic(fmt.Sprintf("Failed to write to file %s\n", filedata.filename))
			}
			wrote += uint32(n)
		}

		var p float32 = 100 * float32(frameCount) / float32(filedata.frames)
		if p-progress > 1 {
			log.Printf("Receiving %v: %v%%\n", filedata.filename, p)
			progress = p
		}

		if frameCount == filedata.frames {
			return
		}

	}
}
