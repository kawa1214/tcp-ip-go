## Tutorial

1. Create TUN/TAP

```sh
bash tun_tap.sh
```

2. Run main.go

```sh
sudo go run main.go
```

3. curl execution

```sh
curl --interface tun0 http://10.0.0.2/
```

## Dump Wireshark

1. Packet Monitoring

```sh
sudo tcpdump -i tun0 -w capture.pcap
```


2. Open capture.pcap in wireshark
