package link

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_NO_PI = 0x1000
)

type Packet struct {
	Buf []byte
	N   uintptr
}

type NetDevice interface {
	Close() error
	Read([]byte) (uintptr, error)
	Write([]byte) (uintptr, error)
	Bind()
	IncomingQueue() chan Packet
	OutgoingQueue() chan Packet
}
