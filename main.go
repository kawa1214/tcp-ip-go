package main

import (
	"bufio"
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

		ipHeader, err := ip.Parse(buf[:n])
		if err != nil {
			log.Fatal(err)
		}
		tcpHeader, err := tcp.Parse(buf[ipHeader.IHL*4 : n])
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

func sendAckResponseWithPayload(file *os.File, ipHeader *ip.Header, tcpHeader *tcp.Header, dataLen int) {
	payload := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 15\r\n\r\nHello, World!\r\n")
	payloadLen := len(payload)

	newIPHeader := ip.New(ipHeader.DstIP, ipHeader.SrcIP, tcp.LENGTH+payloadLen)
	ipHeaderPacket := newIPHeader.Marshal()
	newIPHeader.SetChecksum(ipHeaderPacket)
	ipHeaderPacket = newIPHeader.Marshal()

	newTcpHeader := tcp.New(
		tcpHeader.DstPort,
		tcpHeader.SrcPort,
		tcpHeader.AckNum,
		tcpHeader.SeqNum+uint32(dataLen),
		0x18, // SYN-ACKフラグ (PSH: 0x02, ACK: 0x10)
	)
	tcpHeaderPacket := newTcpHeader.Marshal()
	newTcpHeader.SetChecksum(*ipHeader, append(tcpHeaderPacket, payload...))
	tcpHeaderPacket = newTcpHeader.Marshal()

	responsePacket := append(ipHeaderPacket, tcpHeaderPacket...)
	responsePacket = append(responsePacket, payload...)

	_, err := file.Write(responsePacket)
	if err != nil {
		log.Fatal(err)
	}
}

func sendSynAck(file *os.File, ipHeader *ip.Header, tcpHeader *tcp.Header) {
	newIPHeader := ip.New(ipHeader.DstIP, ipHeader.SrcIP, tcp.LENGTH)
	ipHeaderPacket := newIPHeader.Marshal()
	newIPHeader.SetChecksum(ipHeaderPacket)
	ipHeaderPacket = newIPHeader.Marshal()

	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	newTcpHeader := tcp.New(
		tcpHeader.DstPort,
		tcpHeader.SrcPort,
		uint32(r.Int31()),
		tcpHeader.SeqNum+1,
		0x12, // SYN-ACKフラグ (SYN: 0x02, ACK: 0x10)
	)
	tcpHeaderPacket := newTcpHeader.Marshal()
	newTcpHeader.SetChecksum(*ipHeader, tcpHeaderPacket)
	tcpHeaderPacket = newTcpHeader.Marshal()

	// IPヘッダとTCPヘッダを結合
	synAckPacket := append(ipHeaderPacket, tcpHeaderPacket...)

	// SYN-ACKパケットを送信
	_, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, file.Fd(), uintptr(unsafe.Pointer(&synAckPacket[0])), uintptr(len(synAckPacket)))
	if sysErr != 0 {
		log.Fatalf("Failed to send SYN-ACK packet: %s", sysErr.Error())
	} else {
		log.Printf("SYN-ACK packet sent")
	}
}

func sendFinAck(file *os.File, ipHeader *ip.Header, tcpHeader *tcp.Header, dataLength int) {
	newIPHeader := ip.New(ipHeader.DstIP, ipHeader.SrcIP, tcp.LENGTH)
	ipHeaderPacket := newIPHeader.Marshal()
	newIPHeader.SetChecksum(ipHeaderPacket)
	ipHeaderPacket = newIPHeader.Marshal()

	newTcpHeader := tcp.New(
		tcpHeader.DstPort,
		tcpHeader.SrcPort,
		tcpHeader.AckNum,
		tcpHeader.SeqNum+uint32(dataLength),
		0x11, // SYN-ACKフラグ (FIN, ACK)
	)
	tcpHeaderPacket := newTcpHeader.Marshal()
	newTcpHeader.SetChecksum(*ipHeader, tcpHeaderPacket)
	tcpHeaderPacket = newTcpHeader.Marshal()

	// IPヘッダとTCPヘッダを結合
	synAckPacket := append(ipHeaderPacket, tcpHeaderPacket...)

	// ACKパケットを送信
	_, _, sysErr := syscall.Syscall(syscall.SYS_WRITE, file.Fd(), uintptr(unsafe.Pointer(&synAckPacket[0])), uintptr(len(synAckPacket)))
	if sysErr != 0 {
		log.Fatalf("Failed to send SYN-ACK packet: %s", sysErr.Error())
	} else {
		log.Printf("SYN-ACK packet sent")
	}
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
