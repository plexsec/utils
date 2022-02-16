.PHONY: all clean

all: testdata docker/testdata docker/agent

clean:
	$(RM) testdata docker/testdata docker/agent

testdata:
	go build -o $@

docker/testdata:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@

docker/agent:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ gf.com.cn/gflib/log/agent

IMAGE=docker.gf.com.cn/quote/log_test
build:
	docker build -t $(IMAGE) docker

push:
	docker push $(IMAGE)

