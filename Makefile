# Docker
build:
	docker-compose build --progress=plain --no-cache
up:
	docker-compose up -d
down:
	docker-compose down
remove:
	docker-compose down --remove-orphans

# TCP/IP
tuntap:
	ip tuntap add mode tun dev tun0 &&\
	ip link set tun0 up &&\
	ip addr add 10.0.0.1/24 dev tun0
run:
	go run main.go
curl:
	curl --interface tun0 http://10.0.0.2/

# Wireshark
capture:
	tcpdump -i tun0 -w wireshark/capture.pcap