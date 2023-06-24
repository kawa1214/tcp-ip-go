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
	file   *os.File
	ifreq  *ifreq
	packet chan *packet
}

type packet struct {
	buf []byte
	n   uintptr
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
func (t *Tun) Bind() {
	packets := make(chan *packet, 10)
	go func() {
		for {
			buf := make([]byte, 2048)
			n, err := t.Read(buf)
			if err != nil {
				log.Printf("read error: %s", err.Error())
			}
			log.Printf("read: %d", n)
			packet := &packet{
				buf: buf,
				n:   n,
			}
			packets <- packet
		}
	}()
}
