build:
	go build -ldflags "-s" -o bin/ .
	go build -ldflags "-s" -o bin/ cli/ugotool.go