package link

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_NO_PI = 0x1000
)

type NetDevice interface {
	Close() error
	Read([]byte) (uintptr, error)
	Write([]byte) (uintptr, error)
	Bind()
}
