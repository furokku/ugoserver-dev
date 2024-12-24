build:
	go build -tags "postgres sqlite" -ldflags "-s" -o bin/

postgres:
	go build -tags "postgres" -ldflags "-s" -o bin/

sqlite:
	go build -tags "sqlite" -ldflags "-s" -o bin/