build:
	go build -ldflags "-s" -o bin/ .
	go build -ldflags "-s" -o bin/ sctl/client.go