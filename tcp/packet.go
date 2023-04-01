package tcp

import (
	"encoding/binary"
	"fmt"

	"github.com/kawa1214/tcp-ip-go/ip"
)

const (
	LENGTH      = 20
	WINDOW_SIZE = 65535
	PROTOCOL    = 6
)

type Header struct {
	SrcPort    uint16
	DstPort    uint16
	SeqNum     uint32
	AckNum     uint32
	DataOff    uint8
	Reserved   uint8
	Flags      uint8
	WindowSize uint16
	Checksum   uint16
	UrgentPtr  uint16
}

// New creates a new TCP header from packet.
func Parse(pkt []byte) (*Header, error) {
	if len(pkt) < 20 {
		return nil, fmt.Errorf("invalid TCP header length")
	}

	header := &Header{
		SrcPort:    binary.BigEndian.Uint16(pkt[0:2]),
		DstPort:    binary.BigEndian.Uint16(pkt[2:4]),
		SeqNum:     binary.BigEndian.Uint32(pkt[4:8]),
		AckNum:     binary.BigEndian.Uint32(pkt[8:12]),
		DataOff:    pkt[12] >> 4,
		Reserved:   pkt[12] & 0x0E,
		Flags:      pkt[13],
		WindowSize: binary.BigEndian.Uint16(pkt[14:16]),
		Checksum:   binary.BigEndian.Uint16(pkt[16:18]),
		UrgentPtr:  binary.BigEndian.Uint16(pkt[18:20]),
	}

	return header, nil
}

// Create a new TCP header.
func New(srcPort, dstPort uint16, seqNum, ackNum uint32, flags uint8) *Header {
	return &Header{
		SrcPort:    srcPort,
		DstPort:    dstPort,
		SeqNum:     seqNum,
		AckNum:     ackNum,
		DataOff:    0x50,
		Reserved:   0x12,
		Flags:      flags,
		WindowSize: uint16(WINDOW_SIZE),
		Checksum:   0,
		UrgentPtr:  0,
	}
}

// Return a byte slice of the packet.
func (h *Header) Marshal() []byte {
	pkt := make([]byte, 20)
	binary.BigEndian.PutUint16(pkt[0:2], h.SrcPort)
	binary.BigEndian.PutUint16(pkt[2:4], h.DstPort)
	binary.BigEndian.PutUint32(pkt[4:8], h.SeqNum)
	binary.BigEndian.PutUint32(pkt[8:12], h.AckNum)
	pkt[12] = h.DataOff
	pkt[13] = h.Flags
	binary.BigEndian.PutUint16(pkt[14:16], h.WindowSize)
	binary.BigEndian.PutUint16(pkt[16:18], h.Checksum)
	binary.BigEndian.PutUint16(pkt[18:20], h.UrgentPtr)

	return pkt
}

// Calculates the checksum of the packet and sets Header.
func (h *Header) SetChecksum(ipHeader ip.Header, pkt []byte) {
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], ipHeader.SrcIP[:])
	copy(pseudoHeader[4:8], ipHeader.DstIP[:])
	pseudoHeader[8] = 0
	pseudoHeader[9] = PROTOCOL
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(pkt)))

	buf := append(pseudoHeader, pkt...)
	if len(buf)%2 != 0 {
		buf = append(buf, 0)
	}

	var checksum uint32
	for i := 0; i < len(buf); i += 2 {
		checksum += uint32(binary.BigEndian.Uint16(buf[i : i+2]))
	}

	for checksum > 0xffff {
		checksum = (checksum & 0xffff) + (checksum >> 16)
	}

	h.Checksum = ^uint16(checksum)
}
