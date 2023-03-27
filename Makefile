run:
	go run main.go

kill:
	fuser -kvn tcp 8080

build:
	go build -o bin/main main.go

dtruss:
	sudo dtruss ./bin/main

build-and-dtruss:
	make build
	make dtruss

.PHONY: run build dtruss build-and-dtruss