package main

import (
	"log"

	"github.com/kawa1214/tcp-ip-go/link"
	"github.com/kawa1214/tcp-ip-go/network"
	"github.com/kawa1214/tcp-ip-go/transport"
)

func main() {
	d, err := link.NewTun()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	d.Bind()

	ipPacketQueue := network.NewIpPacketQueue()
	ipPacketQueue.QueuePacket(d)

	tcpPacketQueue := transport.NewTcpPacketQueue()
	tcpPacketQueue.QueuePacket(ipPacketQueue)

	for {

	}

	// s := net.NewStateManager()
	// go s.Listen(d)

	// for {
	// 	conn := s.Accept()
	// 	log.Printf("Accept")

	// 	pkt := conn.Pkt
	// 	n := conn.N
	// 	tcpDataLen := int(n) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)
	// 	resp := server.NewTextOkResponse("Hello, World!\r\n")
	// 	payload := resp.String()
	// 	respNewIPHeader := network.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, transport.LENGTH+len(payload))
	// 	respNewTcpHeader := transport.New(
	// 		pkt.TcpHeader.DstPort,
	// 		pkt.TcpHeader.SrcPort,
	// 		pkt.TcpHeader.AckNum,
	// 		pkt.TcpHeader.SeqNum+uint32(tcpDataLen),
	// 		transport.HeaderFlags{
	// 			PSH: true,
	// 			ACK: true,
	// 		},
	// 	)
	// 	server.Send(d, pkt, &server.TcpPacket{IpHeader: respNewIPHeader, TcpHeader: respNewTcpHeader}, []byte(payload))
	// }
}
