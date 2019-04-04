all: build

REPO_NAME=descheduler

build:
	CGO_ENABLED=0 go build -o bin/${REPO_NAME} github.com/lentil1016/${REPO_NAME}

docker-build:
	docker run -it --rm -v$${PWD}/bin:/go/bin golang:1.11.5 /bin/bash -c \
		"CGO_ENABLED=0 go get -v github.com/lentil1016/${REPO_NAME}"

build-image:
	docker run -it --rm -v$${PWD}/docker:/go/bin golang:1.11.5 /bin/bash -c \
		"CGO_ENABLED=0 go get -v github.com/lentil1016/${REPO_NAME}"
	docker build -t lentil1016/${REPO_NAME} docker

clean:
	rm -f bin/${REPO_NAME} docker/${REPO_NAME}
