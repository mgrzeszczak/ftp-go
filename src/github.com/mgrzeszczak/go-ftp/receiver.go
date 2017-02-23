package main

import (
	"encoding/binary"
	"log"
	"net"
)

const (
	type_header uint16 = 1
	type_frame  uint16 = 2
)

type frame struct {
	len     uint32
	ctype   uint16
	id      uint32
	content []byte
}

func packFrame(f *frame) []byte {
	len := 10 + len(f.content)
	out := make([]byte, len)
	binary.BigEndian.PutUint16(out[:2], f.ctype)
	binary.BigEndian.PutUint32(out[2:6], f.len)
	binary.BigEndian.PutUint32(out[6:10], f.id)
	copy(out[10:], f.content)
	return out
}

func handle(conn *net.Conn, id int, stop chan int) {
	frameMap := make(map[uint32]chan *frame)

	readFrame := func() (*frame, error) {
		//log.Println("Reading frame")
		f := frame{}
		err := binary.Read(*conn, binary.BigEndian, &f.ctype)
		if err != nil {
			return nil, err
		}
		//log.Printf("Read type %v\n", f.ctype)
		err = binary.Read(*conn, binary.BigEndian, &f.len)
		if err != nil {
			return nil, err
		}
		//log.Println("Read len")
		err = binary.Read(*conn, binary.BigEndian, &f.id)
		if err != nil {
			return nil, err
		}
		//log.Println("Read id")
		f.content = make([]byte, f.len)
		err = binary.Read(*conn, binary.BigEndian, f.content)
		if err != nil {
			return nil, err
		}
		//log.Println("Read content")
		//log.Printf("Read frame of type %v\n", f.ctype)
		return &f, nil
	}

	go func() {
		defer func() {
			log.Printf("Closing connection %v\n", (*conn).RemoteAddr().String())
			stop <- id
		}()
		defer func() {
			// close file writers
		}()

		log.Printf("Starting handler for %v\n", (*conn).RemoteAddr().String())
		for {
			f, e := readFrame()
			if e != nil {
				log.Printf("Error reading from %v - %v", (*conn).RemoteAddr().String(), e.Error())
				return
			}
			switch f.ctype {
			case type_header:
				frameMap[f.id] = make(chan *frame, 10)
				go startFileWriter(frameMap[f.id], f)
			case type_frame:
				frameMap[f.id] <- f
			}
		}
	}()
}

func listen(listener net.Listener) <-chan net.Conn {
	c := make(chan net.Conn)
	go func() {
		defer close(c)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			c <- conn
		}
	}()
	return c
}
