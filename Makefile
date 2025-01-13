tools:
	go install github.com/air-verse/air@latest

build:
	go mod tidy
	docker build -t dbot .
