package link

import (
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

type Tun struct {
	file          *os.File
	ifreq         *ifreq
	incomingQueue chan Packet
	outgoingQueue chan Packet
}

// NewTun creates and initializes a new TUN device.
func NewTun() (*Tun, error) {
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

	return &Tun{
		file:  file,
		ifreq: &ifr,
	}, nil
}

// Close closes the TUN device.
func (t *Tun) Close() error {
	return t.file.Close()
}

// Read packets with TUN Device.
func (t *Tun) Read(buf []byte) (uintptr, error) {
	n, _, sysErr := syscall.Syscall(syscall.SYS_READ, t.file.Fd(), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if sysErr != 0 {
		return 0, fmt.Errorf("read error: %s", sysErr.Error())
	}

	return n, nil
}

// Write packets with TUN Device.
func (t *Tun) Write(buf []byte) (uintptr, error) {
	n, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, t.file.Fd(), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if sysErr != 0 {
		return 0, fmt.Errorf("write error: %s", sysErr.Error())
	}

	return n, nil
}

// Bind TUN Device.
func (tun *Tun) Bind() {
	packets := make(chan Packet, 10)
	tun.incomingQueue = packets

	outPackets := make(chan Packet, 10)
	tun.outgoingQueue = outPackets
	go func() {
		for {
			buf := make([]byte, 2048)
			n, err := tun.Read(buf)
			if err != nil {
				log.Printf("read error: %s", err.Error())
			}
			log.Printf("link read: %d", n)
			packet := Packet{
				Buf: buf,
				N:   n,
			}
			packets <- packet
		}
	}()

	go func() {
		for {
			select {
			case pkt := <-tun.outgoingQueue:
				log.Printf("link write: %d", pkt.N)
				_, err := tun.Write(pkt.Buf[:pkt.N])
				if err != nil {
					log.Printf("write error: %s", err.Error())
				}
			}
		}
	}()
}

func (t *Tun) IncomingQueue() chan Packet {
	return t.incomingQueue
}

func (t *Tun) OutgoingQueue() chan Packet {
	return t.outgoingQueue
}
