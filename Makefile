.PHONY: build install clean

build:
	go build -o conductor .

install: build
	mkdir -p ~/.local/bin
	mv conductor ~/.local/bin/

clean:
	rm -f conductor
	rm -rf ~/.conductor
