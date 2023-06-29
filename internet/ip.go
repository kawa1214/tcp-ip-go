package internet

import (
	"context"
	"fmt"
	"log"

	"github.com/kawa1214/tcp-ip-go/network"
)

const (
	QUEUE_SIZE = 10
)

type IpPacket struct {
	IpHeader *Header
	Packet   network.Packet
}

type IpPacketQueue struct {
	incomingQueue chan IpPacket
	outgoingQueue chan network.Packet
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewIpPacketQueue() *IpPacketQueue {
	return &IpPacketQueue{
		incomingQueue: make(chan IpPacket, QUEUE_SIZE),
		outgoingQueue: make(chan network.Packet, QUEUE_SIZE),
	}
}

func (ip *IpPacketQueue) ManageQueues(network *network.NetDevice) {
	ip.ctx, ip.cancel = context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-ip.ctx.Done():
				return
			default:
				pkt, err := network.Read()
				if err != nil {
					log.Printf("read error: %s", err.Error())
				}
				ipHeader, err := unmarshal(pkt.Buf[:pkt.N])
				if err != nil {
					log.Printf("unmarshal error: %s", err)
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
			case pkt := <-ip.outgoingQueue:
				err := network.Write(pkt)
				if err != nil {
					log.Printf("write error: %s", err.Error())
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

func (q *IpPacketQueue) Write(pkt network.Packet) error {
	select {
	case q.outgoingQueue <- pkt:
		return nil
	case <-q.ctx.Done():
		return fmt.Errorf("network closed")
	}
}
