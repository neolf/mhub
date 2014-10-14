package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"

	mqtt "github.com/funkygao/mhub/broker"
	proto "github.com/funkygao/mqttmsg"
)

var host = flag.String("host", "localhost:1883", "hostname of broker")
var id = flag.String("id", "", "client id")
var user = flag.String("user", "", "username")
var pass = flag.String("pass", "", "password")
var dump = flag.Bool("dump", false, "dump messages?")
var conns = flag.Int("conns", 200, "how many conns")

var recv int64

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sub topic [topic topic...]")
		return
	}
	for i := 0; i < flag.NArg(); i++ {
		for j := 0; j < *conns; j++ {
			go subscribe(flag.Arg(i), j)
		}
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for _ = range ticker.C {
			log.Printf("recv %d", atomic.LoadInt64(&recv))
		}
	}()

	<-make(chan bool)
}

func subscribe(topic string, no int) {
	conn, err := net.Dial("tcp", *host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	cc := mqtt.NewClientConn(conn)
	cc.Dump = *dump
	cc.KeepAlive = 5
	cc.ClientId = *id

	if err := cc.Connect(*user, *pass); err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected with client id ", cc.ClientId)

	tq := make([]proto.TopicQos, 1)
	tq[0].Topic = topic
	tq[0].Qos = proto.QosAtMostOnce
	cc.Subscribe(tq)

	for m := range cc.Incoming {
		atomic.AddInt64(&recv, 1)

		if *dump {
			fmt.Print(m.TopicName, "\t")
			m.Payload.WritePayload(os.Stdout)
			fmt.Println("\tr: ", m.Header.Retain)
		}

	}
}
