build:
	go build -ldflags "-s" -o bin/ .
	go build -ldflags "-s" -o bin/ cli/ugotool.go
	
clean:
	rm -f bin/ugoserver*
	rm -f bin/ugotool*