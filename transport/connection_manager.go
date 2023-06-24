package transport

import (
	"log"
	"sync"
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
	SrcPort uint16
	DstPort uint16
	State   State
	// Pkt      *server.TcpPacket
	N        uintptr
	isAccept bool
}

type ConnectionManager struct {
	Connections []Connection
	lock        sync.Mutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		Connections: make([]Connection, 0),
	}
}

func (m *ConnectionManager) recv(pkt TcpPacket) {
	if pkt.TcpHeader.Flags.SYN {
		log.Printf("Received SYN Packet")
		m.addConnection(pkt)

		// TODO: send packet

		m.updateState(pkt, StateSynReceived)
	}
}

func (m *ConnectionManager) addConnection(pkt TcpPacket) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.Connections = append(m.Connections, Connection{
		SrcPort: pkt.TcpHeader.SrcPort,
		DstPort: pkt.TcpHeader.DstPort,
		State:   StateSynReceived,
		N:       pkt.Packet.N,
	})
}

func (m *ConnectionManager) updateState(pkt TcpPacket, state State) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i, conn := range m.Connections {
		if conn.SrcPort == pkt.TcpHeader.SrcPort && conn.DstPort == pkt.TcpHeader.DstPort {
			m.Connections[i].State = state
			return
		}
	}
}
