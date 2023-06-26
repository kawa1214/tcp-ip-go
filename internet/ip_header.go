package internet

import (
	"encoding/binary"
	"fmt"
)

type Header struct {
	Version        uint8
	IHL            uint8
	TOS            uint8
	TotalLength    uint16
	ID             uint16
	Flags          uint8
	FragmentOffset uint16
	TTL            uint8
	Protocol       uint8
	Checksum       uint16
	SrcIP          [4]byte
	DstIP          [4]byte
}

const (
	IP_VERSION        = 4
	IHL               = 5
	TOS               = 0
	TTL               = 64
	LENGTH            = IHL * 4
	TCP_PROTOCOL      = 6
	IP_HEADER_MIN_LEN = 20
)

// New creates a new IP header from packet.
func Parse(pkt []byte) (*Header, error) {
	if len(pkt) < IP_HEADER_MIN_LEN {
		return nil, fmt.Errorf("invalid IP header length")
	}

	header := &Header{
		Version:        pkt[0] >> 4,
		IHL:            pkt[0] & 0x0F,
		TOS:            pkt[1],
		TotalLength:    binary.BigEndian.Uint16(pkt[2:4]),
		ID:             binary.BigEndian.Uint16(pkt[4:6]),
		Flags:          pkt[6] >> 5,
		FragmentOffset: binary.BigEndian.Uint16(pkt[6:8]) & 0x1FFF,
		TTL:            pkt[8],
		Protocol:       pkt[9],
		Checksum:       binary.BigEndian.Uint16(pkt[10:12]),
	}

	copy(header.SrcIP[:], pkt[12:16])
	copy(header.DstIP[:], pkt[16:20])

	return header, nil
}

// Create a new IP header.
func NewIp(srcIP, dstIP [4]byte, len int) *Header {
	return &Header{
		Version:     IP_VERSION,
		IHL:         IHL,
		TOS:         TOS,
		TotalLength: uint16(LENGTH + len),
		ID:          0,
		Flags:       0x40,
		TTL:         64,
		Protocol:    TCP_PROTOCOL,
		Checksum:    0,
		SrcIP:       srcIP,
		DstIP:       dstIP,
	}
}

// Return a byte slice of the packet.
func (h *Header) Marshal() []byte {
	versionAndIHL := (h.Version << 4) | h.IHL
	flagsAndFragmentOffset := (uint16(h.FragmentOffset) << 13) | (h.FragmentOffset & 0x1FFF)

	pkt := make([]byte, 20)
	pkt[0] = versionAndIHL
	pkt[1] = 0
	binary.BigEndian.PutUint16(pkt[2:4], h.TotalLength)
	binary.BigEndian.PutUint16(pkt[4:6], h.ID)
	binary.BigEndian.PutUint16(pkt[6:8], flagsAndFragmentOffset)
	pkt[8] = h.TTL
	pkt[9] = h.Protocol
	binary.BigEndian.PutUint16(pkt[10:12], h.Checksum)
	copy(pkt[12:16], h.SrcIP[:])
	copy(pkt[16:20], h.DstIP[:])

	h.setChecksum(pkt)
	binary.BigEndian.PutUint16(pkt[10:12], h.Checksum)

	return pkt
}

// Calculates the checksum of the packet and sets Header.
func (h *Header) setChecksum(pkt []byte) {
	length := len(pkt)
	var checksum uint32

	for i := 0; i < length; i += 2 {
		checksum += uint32(binary.BigEndian.Uint16(pkt[i : i+2]))
	}

	for checksum > 0xffff {
		checksum = (checksum & 0xffff) + (checksum >> 16)
	}

	h.Checksum = ^uint16(checksum)
}
