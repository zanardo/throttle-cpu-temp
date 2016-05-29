all: build

build:
	go build tct.go

clean:
	rm -f tct

install: build
	sudo install -o root -g root -m 0755 tct /usr/local/bin/tct
	sudo install -o root -g root -m 0644 init/tct.service /etc/systemd/system/tct.service
	sudo systemctl enable tct
	sudo systemctl start tct
