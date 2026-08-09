.PHONY: lint run

run:
	@go build -o txi . && (trap 'go clean; exit' INT TERM EXIT; ./txi main.go) # runs clean no matter how the program exits

lint:
	docker run --rm \
		-v $$(pwd):/app \
		-v ~/.cache/golangci-lint:/root/.cache \
		-w /app \
		golangci/golangci-lint:latest-alpine golangci-lint run -v
