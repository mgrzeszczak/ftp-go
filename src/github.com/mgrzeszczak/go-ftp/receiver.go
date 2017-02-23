package main

import (
	"encoding/binary"
	"log"
	"net"
	"time"
)

type frame struct {
	len     uint32
	ctype   uint8
	content []byte
}

func handle(conn net.Conn, stop chan bool) {
	c := make(chan *message)

	readFrame := func() (*frame, error) {
		f := frame{}
		conn.SetDeadline(time.Now().Add(time.Second))
		err := binary.Read(conn, binary.BigEndian, &f.ctype)
		if err != nil {
			return nil, err
		}
		log.Printf("READ %v\n", f.ctype)
		// timeout only for first read
		conn.SetDeadline(time.Time{})
		err = binary.Read(conn, binary.BigEndian, &f.len)
		if err != nil {
			return nil, err
		}

		f.content = make([]byte, f.len)
		binary.Read(conn, binary.BigEndian, f.content)
		return &f, nil
	}

	go func() {
		defer func() {
			log.Printf("Closing connection %v\n", conn.RemoteAddr().String())
			stop <- true
		}()
		defer close(c)
		log.Printf("Starting handler for %v\n", conn.RemoteAddr().String())
		for {
			select {
			case <-stop:
				log.Printf("Stopping handler for %v\n", conn.RemoteAddr().String())
				return
			default:
				f, e := readFrame()
				if e != nil {
					err, ok := e.(net.Error)
					if ok && err.Timeout() {
						//log.Printf("Timeout from %v\n", conn.RemoteAddr().String())
						continue
					} else if e != nil {
						log.Printf("Error reading from %v - %v", conn.RemoteAddr().String(), e.Error())
						return
					}
				}
				log.Printf("Received frame: %v\n", *f)
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
