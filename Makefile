.PHONY: run build

run: build
	./bin/agones-valheim

build:
	go build -o ./bin/agones-valheim main.go
