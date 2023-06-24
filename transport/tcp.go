package transport

import (
	"log"

	"github.com/kawa1214/tcp-ip-go/network"
)

type TcpPacket struct {
	IpHeader  *network.Header
	TcpHeader *Header
	Packet    network.IpPacket
}

type TcpPacketQueue struct {
	packetChan chan TcpPacket
}

func NewTcpPacketQueue() *TcpPacketQueue {
	return &TcpPacketQueue{}
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
					Packet:    ipPkt,
				}
				log.Printf("tcpPacket flags: %+v", tcpPacket.TcpHeader.Flags)
				packets <- tcpPacket
			}
		}
	}()
}
