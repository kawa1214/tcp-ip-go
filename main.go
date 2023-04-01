package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

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
			p := sendFinAck(tun.File, ipHeader, tcpHeader, tcpDataLength)
			tun.Write(p)
			sendFinAckResponse = true

			os.Exit(0)

		} else if tcpHeader.Flags.SYN {

			log.Printf("SYN packet received")

			// SYN-ACKパケットを送信
			p := sendSynAck(tun.File, ipHeader, tcpHeader)
			tun.Write(p)

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
				tcpDataLength := int(n) - (int(ipHeader.IHL) * 4) - (int(tcpHeader.DataOff) * 4)
				p := sendAckResponseWithPayload(ipHeader, tcpHeader, tcpDataLength)
				tun.Write(p)
				sendHttpRespones = true

				fmt.Println("HTTP response sent")
			}
		}
	}

}

func sendAckResponseWithPayload(ipHeader *ip.Header, tcpHeader *tcp.Header, dataLen int) []byte {
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

	return responsePacket
}

func sendSynAck(file *os.File, ipHeader *ip.Header, tcpHeader *tcp.Header) []byte {
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

	synAckPacket := append(ipHeaderPacket, tcpHeaderPacket...)

	return synAckPacket
}

func sendFinAck(file *os.File, ipHeader *ip.Header, tcpHeader *tcp.Header, dataLength int) []byte {
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

	synAckPacket := append(ipHeaderPacket, tcpHeaderPacket...)

	return synAckPacket
}
