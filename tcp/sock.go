package tcp

import "sync"

type State int

// https://github.com/torvalds/linux/blob/master/include/net/sock.h#L185
const (
	Listen State = iota
	SynReceived
	Established
	CloseWait
	LastAck
	Closed
)

// https://github.com/torvalds/linux/blob/master/include/net/sock.h#L357C1-L357C14
// For simplicity, buffers and timestamps are not implemented.
type Sock struct {
	host  string
	port  uint16
	state State
	lock  sync.Mutex
}

func NewSock(host string, port uint16) *Sock {
	return &Sock{
		host:  host,
		port:  port,
		state: Listen,
	}
}

func StateChange(s *Sock, state State) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.state = state
}

type SockMap struct {
	m    map[uint32]*Sock
	lock sync.Mutex
}

func NewSockMap() *SockMap {
	return &SockMap{
		m: make(map[uint32]*Sock),
	}
}

// https://github.com/torvalds/linux/blob/master/include/net/inet_connection_sock.h#L83C1-L140C3
// https://github.com/torvalds/linux/blob/master/include/net/inet_hashtables.h#L130
