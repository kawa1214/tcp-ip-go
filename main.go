package main

import (
	"log"

	"github.com/kawa1214/tcp-ip-go/link"
	"github.com/kawa1214/tcp-ip-go/net"
	"github.com/kawa1214/tcp-ip-go/network"
	"github.com/kawa1214/tcp-ip-go/server"
	"github.com/kawa1214/tcp-ip-go/tcp"
)

func main() {
	d, err := link.NewTun()
	s := net.NewStateManager()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	go s.Listen(d)

	for {
		conn := s.Accept()
		log.Printf("Accept")

		pkt := conn.Pkt
		n := conn.N
		tcpDataLen := int(n) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)
		resp := server.NewTextOkResponse("Hello, World!\r\n")
		payload := resp.String()
		respNewIPHeader := network.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH+len(payload))
		respNewTcpHeader := tcp.New(
			pkt.TcpHeader.DstPort,
			pkt.TcpHeader.SrcPort,
			pkt.TcpHeader.AckNum,
			pkt.TcpHeader.SeqNum+uint32(tcpDataLen),
			tcp.HeaderFlags{
				PSH: true,
				ACK: true,
			},
		)
		server.Send(d, pkt, &server.TcpPacket{IpHeader: respNewIPHeader, TcpHeader: respNewTcpHeader}, []byte(payload))
	}
}
