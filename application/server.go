package application

import (
	"fmt"

	"github.com/kawa1214/tcp-ip-go/internet"
	"github.com/kawa1214/tcp-ip-go/network"
	"github.com/kawa1214/tcp-ip-go/transport"
)

type Server struct {
	network        *network.NetDevice
	ipPacketQueue  *internet.IpPacketQueue
	tcpPacketQueue *transport.TcpPacketQueue
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) ListenAndServe() error {
	network, err := network.NewTun()
	network.Bind()
	s.network = network
	if err != nil {
		return err
	}
	s.serve()
	return nil
}

func (s *Server) serve() {
	ipPacketQueue := internet.NewIpPacketQueue()
	ipPacketQueue.ManageQueues(s.network)
	s.ipPacketQueue = ipPacketQueue

	tcpPacketQueue := transport.NewTcpPacketQueue()
	tcpPacketQueue.ManageQueues(ipPacketQueue)
	s.tcpPacketQueue = tcpPacketQueue
}

func (s *Server) Close() {
	s.network.Close()
	s.ipPacketQueue.Close()
	s.tcpPacketQueue.Close()
}

func (s *Server) Accept() (transport.Connection, error) {
	conn, err := s.tcpPacketQueue.ReadAcceptConnection()
	if err != nil {
		return transport.Connection{}, fmt.Errorf("accept error: %s", err)
	}

	return conn, nil
}

func (s *Server) Write(conn transport.Connection, resp *HttpResponse) {
	s.tcpPacketQueue.Write(conn, transport.HeaderFlags{
		PSH: true,
		ACK: true,
	},
		[]byte(resp.String()),
	)
}
