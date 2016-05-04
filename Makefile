build: build-linux build-darwin build-windows

build-linux:
	mkdir -p bin/linux-amd64
	GOOS=linux GOARCH=amd64 go build -o bin/linux-amd64 .

build-darwin:
	mkdir -p bin/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build -o bin/darwin-amd64 .

build-windows:
	mkdir -p bin/windows-amd64
	GOOS=windows GOARCH=amd64 go build -o bin/windows-amd64 .
