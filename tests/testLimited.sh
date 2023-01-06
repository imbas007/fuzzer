#!/usr/bin/env bash

cd ..

go run main.go \
    -maxReqSec 17 \
    -w wordlists/big.txt \
    -u https://google.com/FUZZ \
    -fc 403,404 \
    -maxTime 120 \
    -o tmp/test.json \
    -X GET
