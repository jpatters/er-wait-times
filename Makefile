.PHONY: docker
docker:
	docker build -t jpatters/ermon:latest .

.PHONY: build
build:
	go build -o ermon .
