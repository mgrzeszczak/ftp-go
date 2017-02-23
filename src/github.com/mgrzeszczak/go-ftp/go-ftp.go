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
	conn, err := net.Dial(conn_type, host)
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		log.Printf("Disconnecting %v", host)
		conn.Close()
	}()

	log.Printf("Connected to %v", host)

	m := make(map[string]<-chan bool)

	for _, filename := range files {
		done := make(chan bool)
		m[filename] = done
		go sendFile(filename, &conn, done)
		defer close(done)
	}

	for k, v := range m {
		result := <-v
		if result {
			log.Printf("Sent file %s\n", k)
		} else {
			log.Printf("Failed to send file %s\n", k)
		}
	}
}
