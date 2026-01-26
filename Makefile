.PHONY: build build-all test clean docker

build:
	go build -o bin/server ./cmd/server

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.txt coverage.html

docker:
	docker build -t gcp-iam-emulator:latest .

run:
	./bin/server

run-bg:
	./bin/server &
