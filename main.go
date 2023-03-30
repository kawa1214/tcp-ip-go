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

	"github.com/kawa1214/tcp-ip-go/socket"
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

func ParseIPHeader(pkt []byte) (*IPHeader, error) {
	if len(pkt) < 20 {
		return nil, fmt.Errorf("invalid IP header length")
	}

	header := &IPHeader{
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

	// パケットの受信
	buf := make([]byte, 2048)
	for {
		n, err := tun.Read(buf)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		// IPヘッダの解析
		ipHeader, err := ParseIPHeader(buf[:n])
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

func buildIPHeader(srcIP, dstIP [4]byte, protocol uint8, payloadLen int) []byte {
	header := make([]byte, 20)
	header[0] = 0x45                                               // Version (IPv4) and IHL (5)
	header[1] = 0                                                  // TOS
	binary.BigEndian.PutUint16(header[2:4], uint16(20+payloadLen)) // Total Length
	binary.BigEndian.PutUint16(header[4:6], 0)                     // ID
	binary.BigEndian.PutUint16(header[6:8], 0x4000)                // Flags (Don't Fragment) and Fragment Offset
	header[8] = 64                                                 // TTL
	header[9] = protocol                                           // Protocol (TCP)
	binary.BigEndian.PutUint16(header[10:12], 0)                   // Checksum (0で初期化)
	copy(header[12:16], srcIP[:])                                  // Source IP
	copy(header[16:20], dstIP[:])                                  // Destination IP

	// Calculate and set the checksum
	checksum := calculateIPChecksum(header)
	binary.BigEndian.PutUint16(header[10:12], checksum)

	return header
}

func sendAckResponseWithPayload(file *os.File, ipHeader *IPHeader, tcpHeader *TCPHeader, dataLen int) {
	payload := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 15\r\n\r\nHello, World!\r\n")
	payloadLen := len(payload)

	newIPHeader := buildIPHeader(ipHeader.DstIP, ipHeader.SrcIP, 6, 20+payloadLen) // 6: TCP protocol, 20: TCP header length + payload length

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

	responsePacket := append(newIPHeader, newTCPHeader...)
	responsePacket = append(responsePacket, payload...)

	_, err := file.Write(responsePacket)
	if err != nil {
		log.Fatal(err)
	}
}

func sendSynAck(file *os.File, ipHeader *IPHeader, tcpHeader *TCPHeader) {
	newIPHeader := make([]byte, 20)
	newIPHeader[0] = 0x45                                       // Version (IPv4) and IHL (5)
	newIPHeader[1] = 0                                          // TOS
	binary.BigEndian.PutUint16(newIPHeader[2:4], uint16(20+20)) // Total Length
	binary.BigEndian.PutUint16(newIPHeader[4:6], 0)             // ID
	binary.BigEndian.PutUint16(newIPHeader[6:8], 0x4000)        // Flags (Don't Fragment) and Fragment Offset
	newIPHeader[8] = 64                                         // TTL
	newIPHeader[9] = 6                                          // Protocol (TCP)
	binary.BigEndian.PutUint16(newIPHeader[10:12], 0)           // Checksum (0で初期化)
	copy(newIPHeader[12:16], ipHeader.DstIP[:])                 // Source IP
	copy(newIPHeader[16:20], ipHeader.SrcIP[:])                 // Destination IP

	// Calculate and set the checksum
	checksum := calculateIPChecksum(newIPHeader)
	binary.BigEndian.PutUint16(newIPHeader[10:12], checksum)

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
	synAckPacket := append(newIPHeader, newTCPHeader...)

	// SYN-ACKパケットを送信
	_, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, file.Fd(), uintptr(unsafe.Pointer(&synAckPacket[0])), uintptr(len(synAckPacket)))
	if sysErr != 0 {
		log.Fatalf("Failed to send SYN-ACK packet: %s", sysErr.Error())
	} else {
		log.Printf("SYN-ACK packet sent")
	}
}

func sendFinAck(file *os.File, ipHeader *IPHeader, tcpHeader *TCPHeader, dataLength int) {
	newIPHeader := make([]byte, 20)
	newIPHeader[0] = 0x45                                       // Version (IPv4) and IHL (5)
	newIPHeader[1] = 0                                          // TOS
	binary.BigEndian.PutUint16(newIPHeader[2:4], uint16(20+20)) // Total Length
	binary.BigEndian.PutUint16(newIPHeader[4:6], 0)             // ID
	binary.BigEndian.PutUint16(newIPHeader[6:8], 0x4000)        // Flags (Don't Fragment) and Fragment Offset
	newIPHeader[8] = 64                                         // TTL
	newIPHeader[9] = 6                                          // Protocol (TCP)
	binary.BigEndian.PutUint16(newIPHeader[10:12], 0)           // Checksum (0で初期化)
	copy(newIPHeader[12:16], ipHeader.DstIP[:])                 // Source IP
	copy(newIPHeader[16:20], ipHeader.SrcIP[:])                 // Destination IP

	// Calculate and set the checksum
	checksum := calculateIPChecksum(newIPHeader)
	binary.BigEndian.PutUint16(newIPHeader[10:12], checksum)

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
	fmt.Println(len(newIPHeader) + len(newTCPHeader))

	// IPヘッダとTCPヘッダを結合
	synAckPacket := append(newIPHeader, newTCPHeader...)

	// ACKパケットを送信
	_, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, file.Fd(), uintptr(unsafe.Pointer(&synAckPacket[0])), uintptr(len(synAckPacket)))
	if sysErr != 0 {
		log.Fatalf("Failed to send SYN-ACK packet: %s", sysErr.Error())
	} else {
		log.Printf("SYN-ACK packet sent")
	}
}

func calculateIPChecksum(header []byte) uint16 {
	length := len(header)
	var checksum uint32

	for i := 0; i < length; i += 2 {
		checksum += uint32(binary.BigEndian.Uint16(header[i : i+2]))
	}

	for checksum > 0xffff {
		checksum = (checksum & 0xffff) + (checksum >> 16)
	}

	return ^uint16(checksum)
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
