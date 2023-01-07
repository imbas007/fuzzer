#!/usr/bin/env bash

cd ..

go run main.go \
    -w wordlists/big.txt \
    -u http://www.google.com/FUZZ \
    -fc 403,404 \
    -o tmp/standard.json \
    -p http://127.0.0.1:9000 \
    -X GET
