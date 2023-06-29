package main

import (
	"fmt"

	"github.com/kawa1214/tcp-ip-go/internet"
	"github.com/kawa1214/tcp-ip-go/network"
)

func main() {
	network, _ := network.NewTun()
	network.Bind()
	ip := internet.NewIpPacketQueue()
	ip.ManageQueues(network)

	for {
		pkt, _ := ip.Read()
		fmt.Printf("IP Header: %+v\n", pkt.IpHeader)
	}
}
