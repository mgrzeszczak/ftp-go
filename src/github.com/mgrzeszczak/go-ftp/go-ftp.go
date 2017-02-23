package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
)

const (
	port       = "2143"
	local_addr = ":" + port
	conn_type  = "tcp"
	arg_send   = "send"
	arg_recv   = "recv"
)

func main() {
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt)

	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case arg_recv:
		log.Println("Receiving files...")
		err := recv(local_addr, signals)
		if err != nil {
			panic(err)
		}
	case arg_send:
		if len(os.Args) < 4 {
			usage()
			return
		}
		log.Printf("Sending files %v to %v\n", os.Args[3:], os.Args[2])
		send(os.Args[3:], os.Args[2], signals)
	default:
		usage()
	}
}

func usage() {
	fmt.Printf("%s [recv | send] (send?) -> [addr(host:port) file1 file2...]\n", os.Args[0])
}

func recv(addr string, signals chan os.Signal) error {
	listener, err := net.Listen(conn_type, addr)

	if err != nil {
		return err
	}

	defer listener.Close()
	connections := listen(listener)

	var count int
	stop := make(chan int)
	m := make(map[int]*net.Conn)

	for {
		select {
		case k := <-stop:
			delete(m, k)
		case s := <-signals:
			count = len(m)
			for _, v := range m {
				(*v).Close()
			}
			for i := 0; i < count; i++ {
				<-stop
			}
			panic(s)
		case conn := <-connections:
			log.Println("New connection @" + conn.RemoteAddr().String())
			count++
			m[count] = &conn
			go handle(&conn, count, stop)
		}
	}
}

func send(files []string, host string, signals chan os.Signal) {
	var bufferSize int64 = 1024
	filename := files[0]
	f, err := os.Open(filename)
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		f.Close()
	}()

	fi, err := f.Stat()
	if err != nil {
		panic(err.Error())
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

	conn, err := net.Dial(conn_type, host)
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		conn.Close()
	}()

	log.Printf("Connected to %v", host)

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
			n, err := conn.Write(bytes[wrote:])
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

	for {
		select {
		case s := <-signals:
			panic(s)
		default:
			chunk := readChunk()
			//log.Printf("Read chunk: \n\n %v \n\nSending...", chunk)
			if len(chunk) == 0 {
				return
			}

			chunkFrame := frame{
				uint32(len(chunk)),
				type_frame,
				1,
				chunk,
			}

			send(packFrame(&chunkFrame))
			//log.Println("Sent")
		}
	}
}
