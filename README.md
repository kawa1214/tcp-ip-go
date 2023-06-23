## Tutorial

1. Build docker image

```sh
make build
```

2. Run docker container

```sh
make up
```

3. Run main.go(in docker container)

```sh
make serve
```

4. curl execution(in docker container)

```sh
make curl
```

## Dump TCP packets using Wireshark

1. Packet Monitoring(in docker container)

```sh
make capture
```

2. Open wireshark/capture.pcap in wireshark
