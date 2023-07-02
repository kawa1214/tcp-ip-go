package transport

import (
	"context"
	"fmt"
	"log"

	"github.com/kawa1214/tcp-ip-go/internet"
	"github.com/kawa1214/tcp-ip-go/network"
)

const (
	QUEUE_SIZE = 10
)

type TcpPacket struct {
	IpHeader  *internet.Header
	TcpHeader *Header
	Packet    network.Packet
}

type TcpPacketQueue struct {
	manager       *ConnectionManager
	outgoingQueue chan network.Packet
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewTcpPacketQueue() *TcpPacketQueue {
	ConnectionManager := NewConnectionManager()
	return &TcpPacketQueue{
		manager:       ConnectionManager,
		outgoingQueue: make(chan network.Packet, QUEUE_SIZE),
	}
}

func (tcp *TcpPacketQueue) ManageQueues(ip *internet.IpPacketQueue) {
	tcp.ctx, tcp.cancel = context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-tcp.ctx.Done():
				return
			default:
				ipPkt, err := ip.Read()
				if err != nil {
					log.Printf("read error: %s", err.Error())
				}
				tcpHeader, err := unmarshal(ipPkt.Packet.Buf[ipPkt.IpHeader.IHL*4 : ipPkt.Packet.N])
				if err != nil {
					log.Printf("unmarshal error: %s", err)
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
			case <-tcp.ctx.Done():
				return
			case pkt := <-tcp.outgoingQueue:
				err := ip.Write(pkt)
				if err != nil {
					log.Printf("write error: %s", err.Error())
				}
			}
		}
	}()
}

func (tcp *TcpPacketQueue) Close() {
	tcp.cancel()
}

func (tcp *TcpPacketQueue) Write(conn Connection, flgs HeaderFlags, data []byte) {
	pkt := conn.Pkt
	tcpDataLen := int(pkt.Packet.N) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)

	incrementAckNum := 0
	if tcpDataLen == 0 {
		incrementAckNum = 1
	} else {
		incrementAckNum = tcpDataLen
	}
	ackNum := pkt.TcpHeader.SeqNum + uint32(incrementAckNum)

	seqNum := conn.initialSeqNum + conn.incrementSeqNum

	writeIpHdr := internet.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, LENGTH+len(data))
	writeTcpHdr := New(
		pkt.TcpHeader.DstPort,
		pkt.TcpHeader.SrcPort,
		seqNum,
		ackNum,
		flgs,
	)

	ipHdr := writeIpHdr.Marshal()
	tcpHdr := writeTcpHdr.Marshal(conn.Pkt.IpHeader, data)

	writePkt := append(ipHdr, tcpHdr...)
	writePkt = append(writePkt, data...)

	incrementSeqNum := 0
	if flgs.SYN || flgs.FIN {
		incrementSeqNum += 1
	}
	incrementSeqNum += len(data)
	tcp.manager.updateIncrementSeqNum(pkt, uint32(incrementSeqNum))

	tcp.outgoingQueue <- network.Packet{
		Buf: writePkt,
		N:   uintptr(len(writePkt)),
	}
}

func (tcp *TcpPacketQueue) ReadAcceptConnection() (Connection, error) {
	pkt, ok := <-tcp.manager.AcceptConnectionQueue
	if !ok {
		return Connection{}, fmt.Errorf("connection queue is closed")
	}

	return pkt, nil
}
