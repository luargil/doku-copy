B=$(shell git rev-parse --abbrev-ref HEAD)
BRANCH=$(subst /,-,$(B))
GITREV=$(shell git describe --abbrev=7 --always --tags)
REV=$(GITREV)-$(BRANCH)-$(shell date +%Y%m%d-%H:%M:%S)
BIN=doku

UNAME_S:=$(shell uname -s)
GOOS:=
ifeq ($(UNAME_S),Darwin)
	GOOS=darwin
else
	GOOS=linux
endif

build: info
	- @go mod tidy
	- cd app && GOOS=$(GOOS) GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.revision=$(REV) -s -w" -o ../dist/$(BIN)

test:
	- go test -v -timeout=60s -race -mod=vendor -covermode=atomic -coverprofile=coverage.txt ./...

run: build
	- @./dist/$(BIN)

info:
	- @echo "os $(GOOS)"
	- @echo "revision $(REV)"


## React App ##
build-web:
	- cd web/doku && NODE_ENV=production yarn build
	- @rm -rf web/static
	- @cp -r web/doku/build/static web/static
	- @cp web/doku/build/favicon.ico web/static
	- @cp web/doku/build/index.html web/static
	- @cp web/doku/build/manifest.json web/static

clean:
	- @rm -rf web/static

## Docker ##
docker:
	docker build -t amerkurev/$(BIN):master --progress=plain .

docker-run: docker
	docker run --rm -p 9090:9090 --name $(BIN) amerkurev/$(BIN):master

.PHONY: info build build-web clean test docker dist
