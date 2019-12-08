test:
	go test ./limiter -coverprofile=coverage.out

coverage: test
	go tool cover -html=coverage.out

.PHONY: example
example:
	go run example/main.go

integration:
	go test ./tests -count=1
