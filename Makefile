build:
	go build -ldflags "-s" -o bin/ .
	go build -ldflags "-s" -o bin/ cli/ugotool.go cli/viewer.go
	
test:
	go test -vet=off .
	
clean:
	rm -f bin/ugoserver*
	rm -f bin/ugotool*