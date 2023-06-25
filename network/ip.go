package network

import (
	"context"
	"fmt"
	"log"

	"github.com/kawa1214/tcp-ip-go/link"
)

const (
	QUEUE_SIZE = 10
)

type IpPacket struct {
	IpHeader *Header
	Packet   link.Packet
}

type IpPacketQueue struct {
	incomingQueue chan IpPacket
	outgoingQueue chan link.Packet
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewIpPacketQueue() *IpPacketQueue {
	return &IpPacketQueue{
		incomingQueue: make(chan IpPacket, QUEUE_SIZE),
		outgoingQueue: make(chan link.Packet, QUEUE_SIZE),
	}
}

func (ip *IpPacketQueue) ManageQueues(device *link.NetDevice) {
	ip.ctx, ip.cancel = context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-ip.ctx.Done():
				return
			default:
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
				ip.incomingQueue <- ipPacket
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ip.ctx.Done():
				return
			default:
				select {
				case pkt := <-ip.outgoingQueue:
					err := device.Write(pkt)
					if err != nil {
						log.Printf("write error: %s", err.Error())
					}
				}
			}
		}
	}()
}

func (q *IpPacketQueue) Close() {
	q.cancel()
}

func (q *IpPacketQueue) Read() (IpPacket, error) {
	pkt, ok := <-q.incomingQueue
	if !ok {
		return IpPacket{}, fmt.Errorf("incoming queue is closed")
	}
	return pkt, nil
}

func (q *IpPacketQueue) Write(pkt link.Packet) error {
	select {
	case q.outgoingQueue <- pkt:
		return nil
	case <-q.ctx.Done():
		return fmt.Errorf("device closed")
	}
}
