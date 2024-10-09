build:
	@go build -o bin/fs

run: build
	@./bin/fs

test:
	@go test ./... -v  # by added @ we don't want to log commands on the terminal