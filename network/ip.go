package network

import (
	"log"

	"github.com/kawa1214/tcp-ip-go/link"
)

type IpPacket struct {
	IpHeader *Header
	Packet   link.Packet
}

type IpPacketQueue struct {
	incomingQueue chan IpPacket
	outgoingQueue chan link.Packet
}

func NewIpPacketQueue() *IpPacketQueue {
	return &IpPacketQueue{}
}

func (ip *IpPacketQueue) ManageQueues(device *link.NetDevice) {
	packets := make(chan IpPacket, 10)
	ip.incomingQueue = packets

	outPackets := make(chan link.Packet, 10)
	ip.outgoingQueue = outPackets
	go func() {
		for {
			pkt, err := device.Read()
			if err != nil {
				log.Printf("read error: %s", err.Error())
			}
			ipHeader, err := Parse(pkt.Buf[:pkt.N])
			if err != nil {
				log.Printf("parse error: %s", err)
				continue
			}
			ipPacket := IpPacket{
				IpHeader: ipHeader,
				Packet:   pkt,
			}
			packets <- ipPacket
		}
	}()

	go func() {
		for {
			select {
			case pkt := <-ip.outgoingQueue:
				err := device.Write(pkt)
				if err != nil {
					log.Printf("write error: %s", err.Error())
				}
			}
		}
	}()
}

func (q *IpPacketQueue) IncomingQueue() chan IpPacket {
	return q.incomingQueue
}

func (q *IpPacketQueue) OutgoingQueue() chan link.Packet {
	return q.outgoingQueue
}
