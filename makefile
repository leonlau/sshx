PLATFORM_LINUX=GOOS=linux GOARCH=amd64
PLATFORM_DARWIN=GOOS=darwin GOARCH=amd64
PLATFORM_DARWIN_ARM64=GOOS=darwin GOARCH=arm64

all: build-linux build-darwin

install:
	@./build.sh install

build-linux:
	@$(PLATFORM_LINUX) go build -ldflags "-s -w" -o sshx-linux-amd64 ./cmd/sshx
	@$(PLATFORM_LINUX) go build -ldflags "-s -w" -o signaling-linux-amd64 ./cmd/signaling



build-darwin:
	@$(PLATFORM_DARWIN) go build -ldflags "-s -w" -o sshx-darwin-arm64 ./cmd/sshx
