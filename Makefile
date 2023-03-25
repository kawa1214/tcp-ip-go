main:
	go run main.go

build:
	go build -o bin/main main.go

dtruss:
	sudo dtruss ./bin/main

build-and-dtruss:
	make build
	make dtruss

.PHONY: main build dtruss build-and-dtruss