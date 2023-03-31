package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/kawa1214/tcp-ip-go/ip"
	"github.com/kawa1214/tcp-ip-go/socket"
	"github.com/kawa1214/tcp-ip-go/tcp"
)

type IPHeader struct {
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

type TCPHeader struct {
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

func ParseTCPHeader(pkt []byte) (*TCPHeader, error) {
	if len(pkt) < 20 {
		return nil, fmt.Errorf("invalid TCP header length")
	}

	header := &TCPHeader{
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

func main() {
	tun, err := socket.NewTun()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer tun.Close()

	sendHttpRespones := false
	sendFinAckResponse := false

	buf := make([]byte, 2048)
	for {
		n, err := tun.Read(buf)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		// IPヘッダの解析
		ipHeader, err := ip.Parse(buf[:n])
		if err != nil {
			log.Fatal(err)
		}

		// TCPヘッダの解析
		tcpHeader, err := ParseTCPHeader(buf[ipHeader.IHL*4 : n])
		if err != nil {
			log.Fatal(err)
		}

		log.Println("flags:", tcpHeader.Flags)

		// FIN ACKフラグを持っていることを確認
		if tcpHeader.Flags == 0x11 {
			log.Printf("FIN ACK packet received")
			tcpDataLength := int(n) - (int(ipHeader.IHL) * 4) - (int(tcpHeader.DataOff) * 4)
			sendFinAck(tun.File, ipHeader, tcpHeader, tcpDataLength)

			sendFinAckResponse = true

			os.Exit(0)

			// time.Sleep(100 * time.Millisecond)
			// SYNフラグを持っていることを確認
		} else if tcpHeader.Flags == 0x02 {
			// sleep milli
			// time.Sleep(10 * time.Millisecond)

			log.Printf("SYN packet received")

			// SYN-ACKパケットを送信
			sendSynAck(tun.File, ipHeader, tcpHeader)

			// ACK packet check
		} else if tcpHeader.Flags&0x10 != 0 {
			log.Printf("ACK packet received")

			if sendHttpRespones {
				continue
			}

			if sendFinAckResponse {
				continue
			}

			// time.Sleep(100 * time.Millisecond)
			req, err := parseHTTPRequest(string(buf[ipHeader.IHL*4+tcpHeader.DataOff*4:]))
			if err != nil {
				continue
			}
			log.Printf("HTTP request: %s %s %v", req.Method, req.URI, string(buf[ipHeader.IHL*4+tcpHeader.DataOff*4:]))
			if req.Method == "GET" && req.URI == "/" {
				// ACKパケットをGET Req(PSH,ACK)の応答として返す
				tcpDataLength := int(n) - (int(ipHeader.IHL) * 4) - (int(tcpHeader.DataOff) * 4)
				sendAckResponseWithPayload(tun.File, ipHeader, tcpHeader, tcpDataLength)
				sendHttpRespones = true

				fmt.Println("HTTP response sent")
			}
			// FIN ACK packet check
		}
	}

}

func sendAckResponseWithPayload(file *os.File, ipHeader *ip.Header, tcpHeader *TCPHeader, dataLen int) {
	payload := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 15\r\n\r\nHello, World!\r\n")
	payloadLen := len(payload)

	newIPHeader := ip.New(ipHeader.DstIP, ipHeader.SrcIP, tcp.LENGTH+payloadLen)
	ipHeaderPacket := newIPHeader.Marshal()
	newIPHeader.SetChecksum(ipHeaderPacket)
	ipHeaderPacket = newIPHeader.Marshal()

	newTCPHeader := make([]byte, 20)
	binary.BigEndian.PutUint16(newTCPHeader[0:2], tcpHeader.DstPort)
	binary.BigEndian.PutUint16(newTCPHeader[2:4], tcpHeader.SrcPort)
	binary.BigEndian.PutUint32(newTCPHeader[4:8], tcpHeader.AckNum)
	binary.BigEndian.PutUint32(newTCPHeader[8:12], tcpHeader.SeqNum+uint32(dataLen))
	newTCPHeader[12] = 0x50 // Data offset (5 x 4 bytes)
	newTCPHeader[13] = 0x18 // Flags (PSH ACK)
	binary.BigEndian.PutUint16(newTCPHeader[14:16], tcpHeader.WindowSize)
	binary.BigEndian.PutUint16(newTCPHeader[16:18], 0) // Checksum (0で初期化)
	binary.BigEndian.PutUint16(newTCPHeader[18:20], 0) // Urgent pointer

	tcpChecksum := calculateTCPChecksum(ipHeader.SrcIP, ipHeader.DstIP, append(newTCPHeader, payload...))
	binary.BigEndian.PutUint16(newTCPHeader[16:18], tcpChecksum)

	responsePacket := append(ipHeaderPacket, newTCPHeader...)
	responsePacket = append(responsePacket, payload...)

	_, err := file.Write(responsePacket)
	if err != nil {
		log.Fatal(err)
	}
}

func sendSynAck(file *os.File, ipHeader *ip.Header, tcpHeader *TCPHeader) {
	newIPHeader := ip.New(ipHeader.DstIP, ipHeader.SrcIP, tcp.LENGTH)
	ipHeaderPacket := newIPHeader.Marshal()
	newIPHeader.SetChecksum(ipHeaderPacket)
	ipHeaderPacket = newIPHeader.Marshal()

	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	// TCPヘッダを構築
	newTCPHeader := make([]byte, 20)
	binary.BigEndian.PutUint16(newTCPHeader[0:2], tcpHeader.DstPort)
	binary.BigEndian.PutUint16(newTCPHeader[2:4], tcpHeader.SrcPort)
	binary.BigEndian.PutUint32(newTCPHeader[4:8], uint32(r.Int31())) // random
	binary.BigEndian.PutUint32(newTCPHeader[8:12], tcpHeader.SeqNum+1)
	newTCPHeader[12] = 0x50
	newTCPHeader[13] = 0x12                                        // SYN-ACKフラグ (SYN: 0x02, ACK: 0x10)
	binary.BigEndian.PutUint16(newTCPHeader[14:16], uint16(65535)) // ウィンドウサイズ
	binary.BigEndian.PutUint16(newTCPHeader[16:18], 0)             // チェックサム (0で初期化)
	binary.BigEndian.PutUint16(newTCPHeader[18:20], 0)             // Urgentポインタ

	// チェックサムを計算
	tcpChecksum := calculateTCPChecksum(ipHeader.SrcIP, ipHeader.DstIP, newTCPHeader)
	binary.BigEndian.PutUint16(newTCPHeader[16:18], tcpChecksum)

	// IPヘッダとTCPヘッダを結合
	synAckPacket := append(ipHeaderPacket, newTCPHeader...)

	// SYN-ACKパケットを送信
	_, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, file.Fd(), uintptr(unsafe.Pointer(&synAckPacket[0])), uintptr(len(synAckPacket)))
	if sysErr != 0 {
		log.Fatalf("Failed to send SYN-ACK packet: %s", sysErr.Error())
	} else {
		log.Printf("SYN-ACK packet sent")
	}
}

func sendFinAck(file *os.File, ipHeader *ip.Header, tcpHeader *TCPHeader, dataLength int) {
	newIPHeader := ip.New(ipHeader.DstIP, ipHeader.SrcIP, tcp.LENGTH)
	ipHeaderPacket := newIPHeader.Marshal()
	newIPHeader.SetChecksum(ipHeaderPacket)
	ipHeaderPacket = newIPHeader.Marshal()

	// TCPヘッダを構築
	newTCPHeader := make([]byte, 20)
	binary.BigEndian.PutUint16(newTCPHeader[0:2], tcpHeader.DstPort)
	binary.BigEndian.PutUint16(newTCPHeader[2:4], tcpHeader.SrcPort)
	binary.BigEndian.PutUint32(newTCPHeader[4:8], tcpHeader.AckNum)
	binary.BigEndian.PutUint32(newTCPHeader[8:12], tcpHeader.SeqNum+uint32(dataLength))
	newTCPHeader[12] = 0x50
	newTCPHeader[13] = 0x11                                        // ACKフラグ (ACK,FIN)
	binary.BigEndian.PutUint16(newTCPHeader[14:16], uint16(65535)) // ウィンドウサイズ
	binary.BigEndian.PutUint16(newTCPHeader[16:18], 0)             // チェックサム (0で初期化)
	binary.BigEndian.PutUint16(newTCPHeader[18:20], 0)             // Urgentポインタ

	// チェックサムを計算
	tcpChecksum := calculateTCPChecksum(ipHeader.SrcIP, ipHeader.DstIP, newTCPHeader)
	binary.BigEndian.PutUint16(newTCPHeader[16:18], tcpChecksum)

	// IPヘッダとTCPヘッダを結合
	synAckPacket := append(ipHeaderPacket, newTCPHeader...)

	// ACKパケットを送信
	_, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, file.Fd(), uintptr(unsafe.Pointer(&synAckPacket[0])), uintptr(len(synAckPacket)))
	if sysErr != 0 {
		log.Fatalf("Failed to send SYN-ACK packet: %s", sysErr.Error())
	} else {
		log.Printf("SYN-ACK packet sent")
	}
}

func calculateTCPChecksum(srcIP, dstIP [4]byte, tcpHeader []byte) uint16 {
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP[:])
	copy(pseudoHeader[4:8], dstIP[:])
	pseudoHeader[8] = 0
	pseudoHeader[9] = 6 // TCPプロトコル番号
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(tcpHeader)))

	buf := append(pseudoHeader, tcpHeader...)
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

	return ^uint16(checksum)
}

type HTTPRequest struct {
	Method  string
	URI     string
	Version string
	Headers map[string]string
	Body    string
}

func parseHTTPRequest(rawRequest string) (*HTTPRequest, error) {
	scanner := bufio.NewScanner(strings.NewReader(rawRequest))
	var requestLine string

	if scanner.Scan() {
		requestLine = scanner.Text()
	} else {
		return nil, fmt.Errorf("failed to read request line")
	}

	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line: %s", requestLine)
	}

	request := &HTTPRequest{
		Method:  parts[0],
		URI:     parts[1],
		Version: parts[2],
		Headers: make(map[string]string),
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}

		headerParts := strings.SplitN(line, ": ", 2)
		if len(headerParts) != 2 {
			return nil, fmt.Errorf("invalid header: %s", line)
		}

		request.Headers[headerParts[0]] = headerParts[1]
	}

	if request.Method == "POST" || request.Method == "PUT" {
		for scanner.Scan() {
			request.Body += scanner.Text()
		}
	}

	return request, nil
}
