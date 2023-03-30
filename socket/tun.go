package socket

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_TAP   = 0x0002
	IFF_NO_PI = 0x1000
)

type Ifreq struct {
	IfrName  [16]byte
	IfrFlags int16
}

type Tun struct {
	File  *os.File
	ifreq *Ifreq
}

func NewTun() (*Tun, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open error: %s", err.Error())
	}

	ifr := Ifreq{}
	copy(ifr.IfrName[:], []byte("tun0"))
	ifr.IfrFlags = IFF_TUN | IFF_NO_PI

	_, _, sysErr := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr)))
	if sysErr != 0 {
		return nil, fmt.Errorf("ioctl error: %s", sysErr.Error())
	}

	return &Tun{
		File:  file,
		ifreq: &ifr,
	}, nil
}

func (t *Tun) Close() error {
	return t.File.Close()
}
