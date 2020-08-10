.PHONY: all
all:
	go build .

update:
	go get -u ./... && go mod tidy && git add go.* && go build . && git commit -m 'go get -u ./... && go mod tidy' go.*
