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
		log.Printf("Sending files %v to %v\n", os.Args[3:], os.Args[2])
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
	stop := make(chan bool)

	for {
		select {
		case <-stop:
			count--
		case s := <-signals:
			for i := 0; i < count; i++ {
				stop <- true
				<-stop
			}
			panic(s)
		case conn := <-connections:
			log.Println("New connection @" + conn.RemoteAddr().String())
			count++
			go handle(conn, stop)
		}
	}
}

func send(files []string, signals chan os.Signal) {

}
