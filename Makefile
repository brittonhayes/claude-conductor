.PHONY: build install clean

build:
	go build -o claudes ./cmd/claudes

install: build
	mkdir -p ~/.local/bin
	mv claudes ~/.local/bin/

clean:
	rm -f claudes
	rm -rf ~/.claudes
