all: www agent

frontend:
	cd www && yarn build

agent:
	go build -o kegger *.go

install: agent frontend
	mv ./kegger /usr/local/bin

uninstall:
	rm /usr/local/bin/kegger
