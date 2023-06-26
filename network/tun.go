package network

import (
	"context"
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

type ifreq struct {
	ifrName  [16]byte
	ifrFlags int16
}

const (
	TUNSETIFF   = 0x400454ca
	IFF_TUN     = 0x0001
	IFF_NO_PI   = 0x1000
	PACKET_SIZE = 2048
	QUEUE_SIZE  = 10
)

type Packet struct {
	Buf []byte
	N   uintptr
}

type NetDevice struct {
	file          *os.File
	incomingQueue chan Packet
	outgoingQueue chan Packet
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewTun() (*NetDevice, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open error: %s", err.Error())
	}

	ifr := ifreq{}
	copy(ifr.ifrName[:], []byte("tun0"))
	ifr.ifrFlags = IFF_TUN | IFF_NO_PI

	_, _, sysErr := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr)))
	if sysErr != 0 {
		return nil, fmt.Errorf("ioctl error: %s", sysErr.Error())
	}

	return &NetDevice{
		file:          file,
		incomingQueue: make(chan Packet, QUEUE_SIZE),
		outgoingQueue: make(chan Packet, QUEUE_SIZE),
	}, nil
}

func (t *NetDevice) Close() error {
	err := t.file.Close()
	if err != nil {
		return fmt.Errorf("close error: %s", err.Error())
	}
	t.cancel()

	return nil
}

func (t *NetDevice) read(buf []byte) (uintptr, error) {
	n, _, sysErr := syscall.Syscall(syscall.SYS_READ, t.file.Fd(), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if sysErr != 0 {
		return 0, fmt.Errorf("read error: %s", sysErr.Error())
	}

	return n, nil
}

func (t *NetDevice) write(buf []byte) (uintptr, error) {
	n, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, t.file.Fd(), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if sysErr != 0 {
		return 0, fmt.Errorf("write error: %s", sysErr.Error())
	}

	return n, nil
}

func (tun *NetDevice) Bind() {
	tun.ctx, tun.cancel = context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-tun.ctx.Done():
				return
			default:
				buf := make([]byte, PACKET_SIZE)
				n, err := tun.read(buf)
				if err != nil {
					log.Printf("read error: %s", err.Error())
				}
				packet := Packet{
					Buf: buf[:n],
					N:   n,
				}
				tun.incomingQueue <- packet
			}
		}
	}()

	go func() {
		for {
			select {
			case <-tun.ctx.Done():
				return
			case pkt := <-tun.outgoingQueue:
				_, err := tun.write(pkt.Buf[:pkt.N])
				if err != nil {
					log.Printf("write error: %s", err.Error())
				}
			}
		}
	}()
}

func (t *NetDevice) Read() (Packet, error) {
	pkt, ok := <-t.incomingQueue
	if !ok {
		return Packet{}, fmt.Errorf("incoming queue is closed")
	}
	return pkt, nil
}

func (t *NetDevice) Write(pkt Packet) error {
	select {
	case t.outgoingQueue <- pkt:
		return nil
	case <-t.ctx.Done():
		return fmt.Errorf("device closed")
	}
}
