APP=ca-injector
IMAGE=ca-injector
TAG?=latest
DOCKER_ROOT?=rembik
NAMESPACE=cert-manager

FQTAG=$(DOCKER_ROOT)/$(IMAGE):$(TAG)

SHA=$(shell docker inspect --format "{{ index .RepoDigests 0 }}" $(1))

install:
  curl -LsSO https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
	rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
	rm go1.21.0.linux-amd64.tar.gz
	export PATH=$PATH:/usr/local/go/bin
	go version

test:
	go test ./...

build-go: test
	GOOS=linux CGO_ENABLED=0 go build -o app

build-docker: build-go
	docker build -t $(FQTAG) .

deploy-docker: build-docker
  docker push $(FQTAG)

deploy:
	helm upgrade $(APP) ./charts/ca-injector \
    -n $(NAMESPACE) \
		--create-namespace \
    --set image.repository=$(DOCKER_ROOT)/$(IMAGE) \
		--set image.tag=$(TAG) \
    --wait --wait-for-jobs -i
