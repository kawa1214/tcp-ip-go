package socket

type Socket interface {
	Close() error
	Read([]byte) (int, error)
	Write([]byte) (int, error)
}
