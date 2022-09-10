build:
	go build -o server

test:
	go test ./...

fmt:
	go fmt

lint:
	golangci-lint run