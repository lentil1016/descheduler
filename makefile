all: build

build:
	CGO_ENABLED=0 go build ${LDFLAGS} -o _output/bin/descheduler github.com/lentil1016/descheduler

docker-build
	
