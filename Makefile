build: build-linux build-darwin build-windows

build-linux:
	@echo "[linux/amd64] go build"
	@mkdir -p bin/linux-amd64
	@GOOS=linux GOARCH=amd64 go build -o bin/linux-amd64/pprofdump .
	@gzip -f bin/linux-amd64/pprofdump
	@mv bin/linux-amd64/pprofdump.gz bin/pprofdump-linux-amd64.gz

build-darwin:
	@echo "[darwin/amd64] go build"
	@mkdir -p bin/darwin-amd64
	@GOOS=darwin GOARCH=amd64 go build -o bin/darwin-amd64/pprofdump .
	@gzip -f bin/darwin-amd64/pprofdump
	@mv bin/darwin-amd64/pprofdump.gz bin/pprofdump-darwin-amd64.gz

build-windows:
	@echo "[windows/amd64] go build"
	@mkdir -p bin/windows-amd64
	@GOOS=windows GOARCH=amd64 go build -o bin/windows-amd64/pprofdump.exe .
	@gzip -f bin/windows-amd64/pprofdump.exe
	@mv bin/windows-amd64/pprofdump.exe.gz bin/pprofdump-windows-amd64.exe.gz

clean:
	@rm -rf bin