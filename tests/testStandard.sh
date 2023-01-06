#!/usr/bin/env bash

cd ..

go run main.go \
    -w wordlists/big.txt \
    -u https://google.com/FUZZ \
    -fc 403,404 \
    -maxTime 120 \
    -o tmp/standard.json \
    -X GET
