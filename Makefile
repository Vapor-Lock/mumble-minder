BINARY_NAME="mumble-minder"

build:
	mkdir -p bin
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux main.go

clean:
	go clean
	rm -rf bin
