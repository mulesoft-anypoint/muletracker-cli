TEST?=$$(go list ./... | grep -v 'vendor')
HOSTNAME=anypoint.mulesoft.com
NAMESPACE=automation
NAME=muletracker-cli
BINARY=${NAME}
VERSION=1.0.0-SNAPSHOT
OS_ARCH=darwin_arm64

default: install

build:
	go build -o ${BINARY}

release:
	GOOS=darwin GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_darwin_amd64/${BINARY}
	GOOS=darwin GOARCH=arm64 go build -o ./bin/${BINARY}_${VERSION}_darwin_arm64/${BINARY}
	GOOS=freebsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_freebsd_386/${BINARY}
	GOOS=freebsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_freebsd_amd64/${BINARY}
	GOOS=freebsd GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_freebsd_arm/${BINARY}
	GOOS=linux GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_linux_386/${BINARY}
	GOOS=linux GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_linux_amd64/${BINARY}
	GOOS=linux GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_linux_arm/${BINARY}
	GOOS=openbsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_openbsd_386/${BINARY}
	GOOS=openbsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_openbsd_amd64/${BINARY}
	GOOS=solaris GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_solaris_amd64/${BINARY}
	GOOS=windows GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_windows_386/${BINARY}
	GOOS=windows GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_windows_amd64/${BINARY}

gorelease:
	goreleaser release --rm-dist