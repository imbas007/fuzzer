#!/usr/bin/env bash

cd ..

go run main.go \
    -maxReqSec 17 \
    -w wordlists/big.txt \
    -u https://google.com/FUZZ \
    -fc 403,404 \
    -fl 209 \
    -maxTime 15 \
    -o tmp/limited.json \
    -X GET
