package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/kawa1214/tcp-ip-go/ip"
	"github.com/kawa1214/tcp-ip-go/server"
	"github.com/kawa1214/tcp-ip-go/socket"
	"github.com/kawa1214/tcp-ip-go/tcp"
)

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

		if tcpHeader.Flags.FIN && tcpHeader.Flags.ACK {
			log.Printf("FIN ACK packet received")
			tcpDataLength := int(n) - (int(ipHeader.IHL) * 4) - (int(tcpHeader.DataOff) * 4)
			sendFinAck(tun.File, ipHeader, tcpHeader, tcpDataLength)

			sendFinAckResponse = true

			os.Exit(0)

		} else if tcpHeader.Flags.SYN {

			log.Printf("SYN packet received")

			// SYN-ACKパケットを送信
			sendSynAck(tun.File, ipHeader, tcpHeader)

		} else if tcpHeader.Flags.ACK {
			log.Printf("ACK packet received")

			if sendHttpRespones {
				continue
			}

			if sendFinAckResponse {
				continue
			}

			req, err := server.ParseHTTPRequest(string(buf[ipHeader.IHL*4+tcpHeader.DataOff*4:]))
			if err != nil {
				continue
			}
			if req.Method == "GET" && req.URI == "/" {
				// ACKパケットをGET Req(PSH,ACK)の応答として返す
				tcpDataLength := int(n) - (int(ipHeader.IHL) * 4) - (int(tcpHeader.DataOff) * 4)
				sendAckResponseWithPayload(tun.File, ipHeader, tcpHeader, tcpDataLength)
				sendHttpRespones = true

				fmt.Println("HTTP response sent")
			}
		}
	}

}

func sendAckResponseWithPayload(file *os.File, ipHeader *ip.Header, tcpHeader *tcp.Header, dataLen int) {
	response := server.NewTextOkResponse("Hello, World!\r\n")
	payload := response.String()

	newIPHeader := ip.New(ipHeader.DstIP, ipHeader.SrcIP, tcp.LENGTH+len(payload))
	ipHeaderPacket := newIPHeader.Marshal()
	newIPHeader.SetChecksum(ipHeaderPacket)
	ipHeaderPacket = newIPHeader.Marshal()

	newTcpHeader := tcp.New(
		tcpHeader.DstPort,
		tcpHeader.SrcPort,
		tcpHeader.AckNum,
		tcpHeader.SeqNum+uint32(dataLen),
		tcp.HeaderFlags{
			PSH: true,
			ACK: true,
		},
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
		tcp.HeaderFlags{
			SYN: true,
			ACK: true,
		},
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
		tcp.HeaderFlags{
			FIN: true,
			ACK: true,
		},
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
