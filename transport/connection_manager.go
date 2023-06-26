package transport

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/kawa1214/tcp-ip-go/internet"
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
	Pkt      TcpPacket
	N        uintptr
	isAccept bool
}

type ConnectionManager struct {
	Connections           []Connection
	AcceptConnectionQueue chan Connection
	lock                  sync.Mutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		Connections:           make([]Connection, 0),
		AcceptConnectionQueue: make(chan Connection, QUEUE_SIZE),
	}
}

func (m *ConnectionManager) recv(queue *TcpPacketQueue, pkt TcpPacket) {
	conn, ok := m.find(pkt)
	if ok {
		conn.Pkt = pkt
	}

	if pkt.TcpHeader.Flags.SYN {
		log.Printf("Received SYN Packet")
		m.addConnection(pkt)

		newIPHeader := internet.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, LENGTH)
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))
		newTcpHeader := New(
			pkt.TcpHeader.DstPort,
			pkt.TcpHeader.SrcPort,
			uint32(r.Int31()),
			pkt.TcpHeader.SeqNum+1,
			HeaderFlags{
				SYN: true,
				ACK: true,
			},
		)
		sendPkt := TcpPacket{
			IpHeader:  newIPHeader,
			TcpHeader: newTcpHeader,
		}
		queue.Write(pkt, sendPkt, nil)

		m.update(pkt, StateSynReceived, false)
	}

	if ok && pkt.TcpHeader.Flags.ACK && conn.State == StateSynReceived {
		log.Printf("Received ACK Packet")
		m.update(pkt, StateEstablished, false)
	}

	if ok && pkt.TcpHeader.Flags.PSH && conn.State == StateEstablished {
		log.Printf("Received PSH Packet")

		newIPHeader := internet.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, LENGTH)
		tcpDataLen := int(pkt.Packet.N) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)
		newTcpHeader := New(
			pkt.TcpHeader.DstPort,
			pkt.TcpHeader.SrcPort,
			pkt.TcpHeader.AckNum,
			pkt.TcpHeader.SeqNum+uint32(tcpDataLen),
			HeaderFlags{
				ACK: true,
			},
		)
		sendPkt := TcpPacket{
			IpHeader:  newIPHeader,
			TcpHeader: newTcpHeader,
		}
		queue.Write(pkt, sendPkt, nil)
		m.update(pkt, StateEstablished, true)

		m.AcceptConnectionQueue <- conn
	}

	if ok && pkt.TcpHeader.Flags.FIN && conn.State == StateEstablished {
		log.Printf("Received FIN Packet")

		newIPHeader := internet.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, LENGTH)
		newTcpHeader := New(
			pkt.TcpHeader.DstPort,
			pkt.TcpHeader.SrcPort,
			pkt.TcpHeader.AckNum,
			pkt.TcpHeader.SeqNum+1,
			HeaderFlags{
				ACK: true,
			},
		)
		sendPkt := TcpPacket{
			IpHeader:  newIPHeader,
			TcpHeader: newTcpHeader,
		}
		queue.Write(pkt, sendPkt, nil)
		m.update(pkt, StateCloseWait, false)

		newIPHeader = internet.NewIp(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, LENGTH)
		newTcpHeader = New(
			pkt.TcpHeader.DstPort,
			pkt.TcpHeader.SrcPort,
			pkt.TcpHeader.AckNum,
			pkt.TcpHeader.SeqNum+1,
			HeaderFlags{
				FIN: true,
				ACK: true,
			},
		)
		sendPkt = TcpPacket{
			IpHeader:  newIPHeader,
			TcpHeader: newTcpHeader,
		}
		queue.Write(pkt, sendPkt, nil)
		m.update(pkt, StateLastAck, false)
	}

	if ok && pkt.TcpHeader.Flags.ACK && conn.State == StateLastAck {
		log.Printf("Received ACK Packet")
		m.update(pkt, StateClosed, false)
		m.remove(pkt)
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

func (m *ConnectionManager) remove(pkt TcpPacket) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i, conn := range m.Connections {
		if conn.SrcPort == pkt.TcpHeader.SrcPort && conn.DstPort == pkt.TcpHeader.DstPort {
			m.Connections = append(m.Connections[:i], m.Connections[i+1:]...)
			return
		}
	}
}

func (m *ConnectionManager) find(pkt TcpPacket) (Connection, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, conn := range m.Connections {
		if conn.SrcPort == pkt.TcpHeader.SrcPort && conn.DstPort == pkt.TcpHeader.DstPort {
			return conn, true
		}
	}

	return Connection{}, false
}

func (m *ConnectionManager) update(pkt TcpPacket, state State, isAccept bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i, conn := range m.Connections {
		if conn.SrcPort == pkt.TcpHeader.SrcPort && conn.DstPort == pkt.TcpHeader.DstPort {
			m.Connections[i].Pkt = pkt
			m.Connections[i].State = state
			m.Connections[i].isAccept = isAccept
			return
		}
	}
}
