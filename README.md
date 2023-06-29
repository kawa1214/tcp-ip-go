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
go run example/todo/main.go
```

4. curl execution(in docker container)

```sh
curl --interface tun0 -X POST -H "Content-Type: application/json" -d '{
    "title": "ToDo2"
}' 'http://10.0.0.2/todos'

curl --interface tun0 http://10.0.0.2/todos
```

## Dump TCP packets using Wireshark

1. Packet Monitoring(in docker container)

```sh
make capture
```

2. Open wireshark/capture.pcap in wireshark
