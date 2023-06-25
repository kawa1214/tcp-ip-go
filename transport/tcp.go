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
	manager *ConnectionManager
	// incomingQueue chan TcpPacket
	outgoingQueue chan link.Packet
}

func NewTcpPacketQueue() *TcpPacketQueue {
	ConnectionManager := NewConnectionManager()
	return &TcpPacketQueue{
		manager: ConnectionManager,
	}
}

func (tcp *TcpPacketQueue) ManageQueues(ip *network.IpPacketQueue) {
	packets := make(chan link.Packet, 10)
	tcp.outgoingQueue = packets

	go func() {
		for {
			select {
			case ipPkt := <-ip.IncomingQueue():
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

				tcp.manager.recv(tcp, tcpPacket)
			}
		}
	}()

	go func() {
		for {
			select {
			case pkt := <-tcp.outgoingQueue:
				log.Printf("transport write: %d", pkt.N)
				ip.OutgoingQueue() <- pkt
			}
		}
	}()
}

func (tcp *TcpPacketQueue) Write(from, to TcpPacket, data []byte) {
	log.Printf("Write: %d", to.Packet.N)

	ipHdr := to.IpHeader.Marshal()
	to.IpHeader.SetChecksum(ipHdr)
	ipHdr = to.IpHeader.Marshal()

	tcpHdr := to.TcpHeader.Marshal()
	to.TcpHeader.SetChecksum(*from.IpHeader, append(tcpHdr, data...))
	tcpHdr = to.TcpHeader.Marshal()

	pkt := append(ipHdr, tcpHdr...)
	pkt = append(pkt, data...)

	tcp.outgoingQueue <- link.Packet{
		Buf: pkt,
		N:   uintptr(len(pkt)),
	}
}

func (tcp *TcpPacketQueue) ConnectionQueue() chan Connection {
	return tcp.manager.ConnectionQueue
}
