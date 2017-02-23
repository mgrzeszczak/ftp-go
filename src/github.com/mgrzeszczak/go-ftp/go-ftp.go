package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"reflect"
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
		send(os.Args[3:], fmt.Sprintf("%s:%v", os.Args[2], port), signals)
	default:
		usage()
	}
}

func usage() {
	fmt.Printf("%s [recv | send] (send?) -> [host(e.g. 192.168.0.2) file1 file2...]\n", os.Args[0])
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

	channels := make([]chan bool, len(files))
	stopChannels := make([]chan bool, len(files))

	for i, filename := range files {
		done := make(chan bool)
		stop := make(chan bool)
		m[filename] = done
		go sendFile(filename, &conn, done, stop)
		channels[i] = done
		stopChannels[i] = stop
		defer close(stop)
	}

	cases := make([]reflect.SelectCase, len(channels)+1)
	for i, ch := range channels {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	cases[len(cases)-1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(signals)}

	remaining := len(files)
	for remaining > 0 {

		chosen, value, ok := reflect.Select(cases)
		if !ok {
			// The chosen channel has been closed, so zero out the channel to disable the case
			cases[chosen].Chan = reflect.ValueOf(nil)
			if chosen < len(files) {
				remaining -= 1
			}
			continue
		}

		if chosen == len(files) {
			for _, v := range stopChannels {
				v <- true
			}
		} else {
			if value.Bool() {
				log.Printf("Sent %s\n", files[chosen])
			} else {
				log.Printf("Failed to send %s\n", files[chosen])
				remaining -= 1
			}
		}

	}
}
