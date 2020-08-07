#Makefile helm2detector - Deteting tillers and helm 2 releases
NAME=h2d

default: build

.PHONY: build docker test

build: 
	go build -o bin/$(NAME) ./

test:
	go vet ./...
	go test -v ./...

docker:
	docker build -t $(NAME) --build-arg NAME=$(NAME) . 

docker-push:
	docker build -t registry.example.com/$(NAME) --build-arg NAME=$(NAME) .
	docker push registry.example.com/$(NAME)
