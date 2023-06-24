package net

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/kawa1214/tcp-ip-go/link"
	"github.com/kawa1214/tcp-ip-go/network"
	"github.com/kawa1214/tcp-ip-go/server"
	"github.com/kawa1214/tcp-ip-go/tcp"
)

type State int

const (
	StateListen State = iota
	StateSynReceived
	StateEstablished
	StateCloseWait
	StateLastAck
	StateClosed
)

type Connection struct {
	SrcPort  uint16
	DstPort  uint16
	State    State
	Pkt      *server.TcpPacket
	N        uintptr
	isAccept bool
}

type StateManager struct {
	Connections []Connection
	lock        sync.Mutex
}

func NewStateManager() *StateManager {
	return &StateManager{
		Connections: make([]Connection, 0),
	}
}

func (s *StateManager) Listen(d link.NetDevice) {
	for {
		buf := make([]byte, 2048)
		n, err := d.Read(buf)
		if err != nil {
			panic(err)
		}

		pkt, err := server.Parse(buf, n)
		if err != nil {
			panic(err)
		}

		log.Printf("pkt: %+v", pkt)

		if pkt.TcpHeader.Flags.SYN {
			newIPHeader := network.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH)
			seed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(seed))
			newTcpHeader := tcp.New(
				pkt.TcpHeader.DstPort,
				pkt.TcpHeader.SrcPort,
				uint32(r.Int31()),
				pkt.TcpHeader.SeqNum+1,
				tcp.HeaderFlags{
					SYN: true,
					ACK: true,
				},
			)

			s.addConnection(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort)
			server.Send(d, pkt, &server.TcpPacket{IpHeader: newIPHeader, TcpHeader: newTcpHeader}, nil)
			s.updateState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort, StateSynReceived, pkt, n, false)
		}

		if pkt.TcpHeader.Flags.ACK && s.findState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort) == StateSynReceived {
			s.updateState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort, StateEstablished, pkt, n, false)
		}

		if pkt.TcpHeader.Flags.PSH && pkt.TcpHeader.Flags.ACK && s.findState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort) == StateEstablished {
			newIPHeader := network.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH)
			tcpDataLen := int(n) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)
			newTcpHeader := tcp.New(
				pkt.TcpHeader.DstPort,
				pkt.TcpHeader.SrcPort,
				pkt.TcpHeader.AckNum,
				pkt.TcpHeader.SeqNum+uint32(tcpDataLen),
				tcp.HeaderFlags{
					ACK: true,
				},
			)
			server.Send(d, pkt, &server.TcpPacket{IpHeader: newIPHeader, TcpHeader: newTcpHeader}, nil)

			s.updateState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort, StateEstablished, pkt, n, true)
		}

		if pkt.TcpHeader.Flags.FIN && pkt.TcpHeader.Flags.ACK && s.findState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort) == StateEstablished {
			newIPHeader := network.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH)
			newTcpHeader := tcp.New(
				pkt.TcpHeader.DstPort,
				pkt.TcpHeader.SrcPort,
				pkt.TcpHeader.AckNum,
				pkt.TcpHeader.SeqNum+1,
				tcp.HeaderFlags{
					ACK: true,
				},
			)

			server.Send(d, pkt, &server.TcpPacket{IpHeader: newIPHeader, TcpHeader: newTcpHeader}, nil)
			s.updateState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort, StateCloseWait, pkt, n, false)

			finNewIPHeader := network.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH)
			finNewTcpHeader := tcp.New(
				pkt.TcpHeader.DstPort,
				pkt.TcpHeader.SrcPort,
				pkt.TcpHeader.AckNum,
				pkt.TcpHeader.SeqNum+1,
				tcp.HeaderFlags{
					FIN: true,
					ACK: true,
				},
			)
			server.Send(d, pkt, &server.TcpPacket{IpHeader: finNewIPHeader, TcpHeader: finNewTcpHeader}, nil)
			s.updateState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort, StateLastAck, pkt, n, false)
		}

		if pkt.TcpHeader.Flags.ACK && s.findState(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort) == StateLastAck {
			s.removeConnection(pkt.TcpHeader.DstPort, pkt.TcpHeader.SrcPort)
		}
	}
}

func (s *StateManager) findState(srcPort, dstPort uint16) State {
	for _, conn := range s.Connections {
		if conn.SrcPort == srcPort && conn.DstPort == dstPort {
			return conn.State
		}
	}
	return StateClosed
}

func (s *StateManager) Accept() Connection {
	for {
		for _, conn := range s.Connections {
			if conn.isAccept {
				s.updateState(conn.SrcPort, conn.DstPort, conn.State, conn.Pkt, conn.N, false)
				return conn
			}
		}
	}
}

func (s *StateManager) addConnection(srcPort, dstPort uint16) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Connections = append(s.Connections, Connection{
		SrcPort: srcPort,
		DstPort: dstPort,
		State:   StateListen,
	})
}

func (s *StateManager) removeConnection(srcPort, dstPort uint16) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, conn := range s.Connections {
		if conn.SrcPort == srcPort && conn.DstPort == dstPort {
			s.Connections = append(s.Connections[:i], s.Connections[i+1:]...)
			return
		}
	}
}

func (s *StateManager) updateState(srcPort, dstPort uint16, newState State, pkt *server.TcpPacket, n uintptr, isAccept bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i, conn := range s.Connections {
		if conn.SrcPort == srcPort && conn.DstPort == dstPort {
			s.Connections[i].State = newState
			s.Connections[i].Pkt = pkt
			s.Connections[i].N = n
			s.Connections[i].isAccept = isAccept

			return
		}
	}
}
