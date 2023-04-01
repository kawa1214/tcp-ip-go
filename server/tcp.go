package server

import (
	"github.com/kawa1214/tcp-ip-go/ip"
	"github.com/kawa1214/tcp-ip-go/socket"
	"github.com/kawa1214/tcp-ip-go/tcp"
)

type TcpPacket struct {
	IpHeader  *ip.Header
	TcpHeader *tcp.Header
}

// Parse packet and returns ip and tcp headers.
func Parse(pkt []byte, length uintptr) (*TcpPacket, error) {
	ipHeader, err := ip.Parse(pkt[:length])
	if err != nil {
		return nil, err
	}

	tcpHeader, err := tcp.Parse(pkt[ipHeader.IHL*4 : length])
	if err != nil {
		return nil, err
	}

	return &TcpPacket{
		IpHeader:  ipHeader,
		TcpHeader: tcpHeader,
	}, nil
}

// Send packet.
func Send(socket *socket.Tun, from, to *TcpPacket, data []byte) error {
	ip := to.IpHeader.Marshal()
	to.IpHeader.SetChecksum(ip)
	ip = to.IpHeader.Marshal()

	tcp := to.TcpHeader.Marshal()
	to.TcpHeader.SetChecksum(*from.IpHeader, append(tcp, data...))
	tcp = to.TcpHeader.Marshal()

	pkt := append(ip, tcp...)
	pkt = append(pkt, data...)

	_, err := socket.Write(pkt)
	if err != nil {
		return err
	}

	return nil
}
