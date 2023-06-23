package main

import (
	"context"
	"log"

	"github.com/kawa1214/tcp-ip-go/net"
	"github.com/kawa1214/tcp-ip-go/socket"
)

func main() {
	tun, err := socket.NewTun()
	s := net.NewStateManager()
	if err != nil {
		log.Fatal(err)
	}
	defer tun.Close()

	listenCtx := context.Background()
	go s.Listen(tun, listenCtx)

	for {
		s.Accept()
	}
}
