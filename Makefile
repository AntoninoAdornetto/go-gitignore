test:
	go test -v ./...

lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.56.2 golangci-lint run -v

coverage:
	go clean -testcache && go test -coverprofile=coverage/coverage.out ./... && go tool cover -html=coverage/coverage.out -o=coverage/coverage.html


.PHONY: test lint coverage
