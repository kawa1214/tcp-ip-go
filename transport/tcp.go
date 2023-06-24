package transport

import (
	"log"

	"github.com/kawa1214/tcp-ip-go/link"
	"github.com/kawa1214/tcp-ip-go/network"
)

type TcpPacket struct {
	IpHeader  *network.Header
	TcpHeader *Header
	Packet    link.Packet
}

type TcpPacketQueue struct {
	manager    *ConnectionManager
	packetChan chan TcpPacket
}

func NewTcpPacketQueue() *TcpPacketQueue {
	ConnectionManager := NewConnectionManager()
	return &TcpPacketQueue{
		manager: ConnectionManager,
	}
}

func (q *TcpPacketQueue) QueuePacket(ipPacketQueue *network.IpPacketQueue) {
	packets := make(chan TcpPacket, 10)
	q.packetChan = packets

	go func() {
		for {
			select {
			case ipPkt := <-ipPacketQueue.PacketChan():
				log.Printf("transport pkt: %d", ipPkt.Packet.N)
				tcpHeader, err := Parse(ipPkt.Packet.Buf[ipPkt.IpHeader.IHL*4 : ipPkt.Packet.N])
				if err != nil {
					log.Printf("parse error: %s", err)
					continue
				}
				tcpPacket := TcpPacket{
					IpHeader:  ipPkt.IpHeader,
					TcpHeader: tcpHeader,
					Packet:    ipPkt.Packet,
				}
				packets <- tcpPacket
				q.manager.recv(tcpPacket)
			}
		}
	}()
}
