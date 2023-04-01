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
	}
	defer tun.Close()

	sendHttpRespones := false
	sendFinAckResponse := false

	buf := make([]byte, 2048)
	for {
		n, err := tun.Read(buf)
		if err != nil {
			log.Fatal(err)
		}

		pkt, err := server.Parse(buf, n)
		if err != nil {
			log.Fatal(err)
		}

		if pkt.TcpHeader.Flags.FIN && pkt.TcpHeader.Flags.ACK {
			log.Printf("FIN ACK packet received")
			tcpDataLen := int(n) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)

			newIPHeader := ip.New(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH)
			newTcpHeader := tcp.New(
				pkt.TcpHeader.DstPort,
				pkt.TcpHeader.SrcPort,
				pkt.TcpHeader.AckNum,
				pkt.TcpHeader.SeqNum+uint32(tcpDataLen),
				tcp.HeaderFlags{
					FIN: true,
					ACK: true,
				},
			)

			server.Send(tun, pkt, &server.TcpPacket{IpHeader: newIPHeader, TcpHeader: newTcpHeader}, nil)

			sendFinAckResponse = true

			os.Exit(0)

		} else if pkt.TcpHeader.Flags.SYN {
			log.Printf("SYN packet received")

			newIPHeader := ip.New(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH)
			seed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(seed))
			newTcpHeader := tcp.New(
				pkt.TcpHeader.DstPort,
				pkt.TcpHeader.SrcPort,
				uint32(r.Int31()),
				pkt.TcpHeader.SeqNum+1,
				tcp.HeaderFlags{
					SYN: true,
					ACK: true,
				},
			)
			server.Send(tun, pkt, &server.TcpPacket{IpHeader: newIPHeader, TcpHeader: newTcpHeader}, nil)

		} else if pkt.TcpHeader.Flags.ACK {
			log.Printf("ACK packet received")

			if sendHttpRespones || sendFinAckResponse {
				continue
			}

			req, err := server.ParseHTTPRequest(string(buf[pkt.IpHeader.IHL*4+pkt.TcpHeader.DataOff*4:]))
			if err != nil {
				continue
			}

			if req.Method == "GET" && req.URI == "/" {
				tcpDataLen := int(n) - (int(pkt.IpHeader.IHL) * 4) - (int(pkt.TcpHeader.DataOff) * 4)

				resp := server.NewTextOkResponse("Hello, World!\r\n")
				payload := resp.String()

				newIPHeader := ip.New(pkt.IpHeader.DstIP, pkt.IpHeader.SrcIP, tcp.LENGTH+len(payload))
				newTcpHeader := tcp.New(
					pkt.TcpHeader.DstPort,
					pkt.TcpHeader.SrcPort,
					pkt.TcpHeader.AckNum,
					pkt.TcpHeader.SeqNum+uint32(tcpDataLen),
					tcp.HeaderFlags{
						PSH: true,
						ACK: true,
					},
				)

				server.Send(tun, pkt, &server.TcpPacket{IpHeader: newIPHeader, TcpHeader: newTcpHeader}, []byte(payload))

				sendHttpRespones = true

				fmt.Println("HTTP response sent")
			}
		}
	}
}
