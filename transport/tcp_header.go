package transport

import (
	"encoding/binary"
	"fmt"

	"github.com/kawa1214/tcp-ip-go/internet"
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
	Flags      HeaderFlags
	WindowSize uint16
	Checksum   uint16
	UrgentPtr  uint16
}

type HeaderFlags struct {
	CWR bool
	ECE bool
	URG bool
	ACK bool
	PSH bool
	RST bool
	SYN bool
	FIN bool
}

// New creates a new TCP header from packet.
func Parse(pkt []byte) (*Header, error) {
	if len(pkt) < 20 {
		return nil, fmt.Errorf("invalid TCP header length")
	}

	flags := parseFlag(pkt[13])

	header := &Header{
		SrcPort:    binary.BigEndian.Uint16(pkt[0:2]),
		DstPort:    binary.BigEndian.Uint16(pkt[2:4]),
		SeqNum:     binary.BigEndian.Uint32(pkt[4:8]),
		AckNum:     binary.BigEndian.Uint32(pkt[8:12]),
		DataOff:    pkt[12] >> 4,
		Reserved:   pkt[12] & 0x0E,
		Flags:      flags,
		WindowSize: binary.BigEndian.Uint16(pkt[14:16]),
		Checksum:   binary.BigEndian.Uint16(pkt[16:18]),
		UrgentPtr:  binary.BigEndian.Uint16(pkt[18:20]),
	}

	return header, nil
}

// Create a new TCP header.
func New(srcPort, dstPort uint16, seqNum, ackNum uint32, flags HeaderFlags) *Header {
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
func (h *Header) Marshal(ipHdr *internet.Header, data []byte) []byte {
	f := h.Flags.marshal()

	pkt := make([]byte, 20)
	binary.BigEndian.PutUint16(pkt[0:2], h.SrcPort)
	binary.BigEndian.PutUint16(pkt[2:4], h.DstPort)
	binary.BigEndian.PutUint32(pkt[4:8], h.SeqNum)
	binary.BigEndian.PutUint32(pkt[8:12], h.AckNum)
	pkt[12] = h.DataOff
	pkt[13] = f
	binary.BigEndian.PutUint16(pkt[14:16], h.WindowSize)
	binary.BigEndian.PutUint16(pkt[16:18], h.Checksum)
	binary.BigEndian.PutUint16(pkt[18:20], h.UrgentPtr)

	h.setChecksum(ipHdr, append(pkt, data...))
	binary.BigEndian.PutUint16(pkt[16:18], h.Checksum)

	return pkt
}

// Calculates the checksum of the packet and sets Header.
func (h *Header) setChecksum(ipHeader *internet.Header, pkt []byte) {
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

// Return a string representation of the packet byte.
func parseFlag(f uint8) HeaderFlags {
	return HeaderFlags{
		CWR: f&0x80 == 0x80,
		ECE: f&0x40 == 0x40,
		URG: f&0x20 == 0x20,
		ACK: f&0x10 == 0x10,
		PSH: f&0x08 == 0x08,
		RST: f&0x04 == 0x04,
		SYN: f&0x02 == 0x02,
		FIN: f&0x01 == 0x01,
	}
}

// Return a byte slice of the packet.
func (f *HeaderFlags) marshal() uint8 {
	var packedFlags uint8
	if f.CWR {
		packedFlags |= 0x80
	}
	if f.ECE {
		packedFlags |= 0x40
	}
	if f.URG {
		packedFlags |= 0x20
	}
	if f.ACK {
		packedFlags |= 0x10
	}
	if f.PSH {
		packedFlags |= 0x08
	}
	if f.RST {
		packedFlags |= 0x04
	}
	if f.SYN {
		packedFlags |= 0x02
	}
	if f.FIN {
		packedFlags |= 0x01
	}
	return packedFlags
}
