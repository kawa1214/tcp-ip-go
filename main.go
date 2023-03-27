package main

import (
	"encoding/binary"
	"log"
	"net"
	"strconv"
	"syscall"
)

type IPHeader struct {
	Version                uint8  // バージョン
	HeaderLength           uint8  // ヘッダ長
	DifferentiatedServices uint8  // ディファレンシェーション・サービス
	TotalLength            uint16 // トータル長
	Identification         uint16 // 識別子
	TimeToLive             uint8  // 生存時間
	Protocol               uint8  // プロトコル
	HeaderChecksum         uint16 // チェックサム
	SourceIP               string // 送信元IPアドレス
	DestinationIP          string // 宛先IPアドレス
}

type TCPHeader struct {
	SrcPort       uint16
	DstPort       uint16
	SeqNum        uint32
	AckNum        uint32
	DataOffset    uint8
	Reserved      uint8
	Flags         uint16
	WindowSize    uint16
	Checksum      uint16
	UrgentPointer uint16
}

func main() {
	serverIP := "127.0.0.1"

	// SOCK_RAWでソケットを作成
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		log.Fatalf("Failed to create socket: %v", err)
	}
	defer syscall.Close(fd)

	if err != nil {
		log.Fatalf("Failed to set IP_HDRINCL option: %v", err)
	}

	// 受信ポートを指定
	serverAddr := syscall.SockaddrInet4{
		Port: 8080,
	}
	copy(serverAddr.Addr[:], net.ParseIP(serverIP).To4())
	err = syscall.Bind(fd, &serverAddr)
	if err != nil {
		log.Fatalf("Failed to bind: %v", err)
	}

	// 受信バッファを作成
	buf := make([]byte, 1024)

	for {
		log.Println("Waiting for packet...")
		// パケットを受信
		n, _, err := syscall.Recvfrom(fd, buf, 0)
		if err != nil {
			log.Printf("Failed to receive packet: %v", err)
			continue
		}

		log.Printf("Received %d bytes", n)

		// IPヘッダを解析
		// https://www.infraexpert.com/study/tea11.htm
		length := ((buf[0] << 4) >> 4) * 4 // IHLフィールドからIPヘッダの長さを計算
		var ipHdrBuf = buf[:length]

		var ipHeaderStruct IPHeader
		version := ipHdrBuf[0] >> 4

		var sourceIP string
		var destinationIP string
		for i := 0; i < 4; i++ {
			sourceIP += strconv.Itoa(int(ipHdrBuf[12+i]))
			destinationIP += strconv.Itoa(int(ipHdrBuf[16+i]))
			if i != 3 {
				sourceIP += "."
				destinationIP += "."
			}
		}

		ipHeaderStruct.Version = version
		ipHeaderStruct.HeaderLength = length
		ipHeaderStruct.DifferentiatedServices = ipHdrBuf[1]
		ipHeaderStruct.TotalLength = binary.BigEndian.Uint16(ipHdrBuf[2:4]) // wireshark: 60, this: 40
		ipHeaderStruct.Identification = binary.BigEndian.Uint16(ipHdrBuf[4:6])
		ipHeaderStruct.TimeToLive = ipHdrBuf[8]
		ipHeaderStruct.Protocol = ipHdrBuf[9] // TCP = 6
		ipHeaderStruct.HeaderChecksum = binary.BigEndian.Uint16(ipHdrBuf[10:12])
		ipHeaderStruct.SourceIP = sourceIP
		ipHeaderStruct.DestinationIP = destinationIP

		log.Printf("IP Header: %+v", ipHeaderStruct)

		// TCPヘッダを解析
		// https://www.infraexpert.com/study/tcpip8.html
	}
}
