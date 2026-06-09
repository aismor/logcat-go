.PHONY: build deb clean run

APP := logcat-go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo 1.0.0)

build:
	CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o dist/$(APP) ./cmd/main.go

deb:
	chmod +x scripts/build-deb.sh
	VERSION=$(VERSION) ./scripts/build-deb.sh

run: build
	./dist/$(APP)

clean:
	rm -rf dist/deb-stage dist/$(APP) dist/*.deb
