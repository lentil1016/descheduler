all: build

build:
	CGO_ENABLED=0 go build -o bin/descheduler github.com/lentil1016/descheduler

docker-build:
	docker run -it --rm -v$${PWD}/bin:/go/bin golang:1.11.5 /bin/bash -c \
		"CGO_ENABLED=0 go get -v github.com/lentil1016/descheduler"

build-image:
	docker run -it --rm -v$${PWD}/docker:/go/bin golang:1.11.5 /bin/bash -c \
		"CGO_ENABLED=0 go get -v github.com/lentil1016/descheduler"
	docker build -t lentil1016/descheduler docker
