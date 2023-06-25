package application

import (
	"github.com/kawa1214/tcp-ip-go/link"
	"github.com/kawa1214/tcp-ip-go/network"
	"github.com/kawa1214/tcp-ip-go/transport"
)

type Server struct {
	link           *link.NetDevice
	ipPacketQueue  *network.IpPacketQueue
	tcpPacketQueue *transport.TcpPacketQueue
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) ListenAndServe() error {
	link, err := link.NewTun()
	link.Bind()
	s.link = link
	if err != nil {
		return err
	}
	s.serve()
	return nil
}

func (s *Server) serve() {
	ipPacketQueue := network.NewIpPacketQueue()
	ipPacketQueue.ManageQueues(s.link)
	s.ipPacketQueue = ipPacketQueue

	tcpPacketQueue := transport.NewTcpPacketQueue()
	tcpPacketQueue.ManageQueues(ipPacketQueue)
	s.tcpPacketQueue = tcpPacketQueue
}

func (s *Server) Close() error {
	s.link.Close()
	return nil
}

func (s *Server) Accept() transport.Connection {
	select {
	case conn := <-s.tcpPacketQueue.ConnectionQueue():
		return conn
	}
}

func (s *Server) Write(conn transport.Connection, resp *HTTPResponse) {
	pkt := conn.Pkt
	tcpDataLen := int(pkt.Packet.N) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)

	payload := resp.String()
	respNewIPHeader := network.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, transport.LENGTH+len(payload))
	respNewTcpHeader := transport.New(
		pkt.TcpHeader.DstPort,
		pkt.TcpHeader.SrcPort,
		pkt.TcpHeader.AckNum,
		pkt.TcpHeader.SeqNum+uint32(tcpDataLen),
		transport.HeaderFlags{
			PSH: true,
			ACK: true,
		},
	)
	sendPkt := transport.TcpPacket{
		IpHeader:  respNewIPHeader,
		TcpHeader: respNewTcpHeader,
	}

	s.tcpPacketQueue.Write(pkt, sendPkt, []byte(payload))
}
