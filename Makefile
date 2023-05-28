.PHONY: k3s-installer

k3s-installer:
	CGO_ENABLED=0 go build -o bin/k3s-installer cmd/main.go