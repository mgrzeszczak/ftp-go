package main

import (
	"log"
	"net"
	"os"
)

func sendFile(filename string, conn *net.Conn, done chan<- bool, stop <-chan bool) {
	var bufferSize int64 = 1024
	f, err := os.Open(filename)
	if err != nil {
		log.Println(err.Error())
		done <- false
		return
	}
	defer func() {
		f.Close()
		close(done)
	}()

	fi, err := f.Stat()
	if err != nil {
		log.Println(err.Error())
		done <- false
		return
	}

	var frameCount uint32
	if fi.Size()%bufferSize == 0 {
		frameCount = uint32(fi.Size() / bufferSize)
	} else {
		frameCount = uint32(fi.Size()/bufferSize + 1)
	}

	filedata := fdata{frameCount, filename}

	log.Printf("Prepared file %v\n", filedata)

	fdatabytes := packFileData(&filedata)

	headerFrame := frame{
		uint32(len(fdatabytes)),
		type_header,
		1,
		fdatabytes,
	}

	send := func(bytes []byte) {
		wrote := 0
		l := int(len(bytes))
		for wrote < l {
			n, err := (*conn).Write(bytes[wrote:])
			if err != nil {
				panic(err.Error())
			}
			wrote += n
		}
	}

	readChunk := func() []byte {
		buf := make([]byte, bufferSize)
		var read int
		for read < int(bufferSize) {
			n, err := f.Read(buf[read:])
			if err != nil {
				return buf[:read]
			}
			read += n
		}
		return buf
	}

	//log.Println("Sending header frame")
	//log.Printf("%v\n", packFrame(&headerFrame))
	send(packFrame(&headerFrame))

	chunkCounter := 0

	for {

		select {
		case <-stop:
			done <- false
			return
		default:
			chunk := readChunk()
			//log.Printf("Read chunk: \n\n %v \n\nSending...", chunk)
			if len(chunk) == 0 {
				//log.Printf("Sent file %v\n", filename)
				done <- true
				return
			}

			chunkFrame := frame{
				uint32(len(chunk)),
				type_frame,
				1,
				chunk,
			}
			send(packFrame(&chunkFrame))

			chunkCounter++
			log.Printf("Sending %v: %v%%\n", filename, 100*float32(chunkCounter)/float32(frameCount))
			//log.Println("Sent")
		}

	}

}
