package main

import (
	"log"
	"syscall"
)

func main() {
	// socket.listen()でサーバー側のカーネルを接続待ち状態にする

	// addr localhost:8080
	// socket関数 https://cs.opensource.google/go/go/+/refs/tags/go1.20.2:src/syscall/syscall_unix.go;l=494

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	log.Println("fd", fd)
	if err != nil {
		panic(err)
	}

	// ソケットにアドレスを割り当てる
	// https://linuxjm.osdn.jp/html/LDP_man-pages/man2/bind.2.html
	err = syscall.Bind(fd, &syscall.SockaddrInet4{Port: 8080})

	if err != nil {
		panic(err)
	}

	// socket.listen()でソケットを接続待ち状態にする
	syscall.Listen(fd, 10)

	for {
		// socket.accept()でクライアントからの接続を受け付ける
		nfd, addr, err := syscall.Accept(fd)
		log.Println("nfd", nfd)
		log.Println("addr", addr)
		if err != nil {
			panic(err)
		}

		// クライアントからのリクエストを読み込む
		r := make([]byte, 1024)
		syscall.Read(nfd, r)
		log.Println(string(r))

		// クライアントにレスポンスを書き込む
		syscall.Write(nfd, []byte("HTTP/1.1 200 OK\r, Content-Length: 12\r\n\r\nHello World!"))

		// ソケットを閉じる
		syscall.Close(nfd)
	}
}
