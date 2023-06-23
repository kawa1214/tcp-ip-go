package main

import (
	"log"

	"github.com/kawa1214/tcp-ip-go/ip"
	"github.com/kawa1214/tcp-ip-go/net"
	"github.com/kawa1214/tcp-ip-go/server"
	"github.com/kawa1214/tcp-ip-go/socket"
	"github.com/kawa1214/tcp-ip-go/tcp"
)

func main() {
	tun, err := socket.NewTun()
	s := net.NewStateManager()
	if err != nil {
		log.Fatal(err)
	}
	defer tun.Close()

	go s.Listen(tun)

	for {
		conn := s.Accept()
		log.Printf("Accept")

		pkt := conn.Pkt
		n := conn.N
		tcpDataLen := int(n) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)
		resp := server.NewTextOkResponse("Hello, World!\r\n")
		payload := resp.String()
		respNewIPHeader := ip.New(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH+len(payload))
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
		server.Send(tun, pkt, &server.TcpPacket{IpHeader: respNewIPHeader, TcpHeader: respNewTcpHeader}, []byte(payload))
	}
}
