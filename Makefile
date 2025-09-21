.PHONY: build

BINARY = sbctl
ifeq ($(OS),Windows_NT)
    BINARY := $(BINARY).exe
endif

build: 
	go build -o $(BINARY)
