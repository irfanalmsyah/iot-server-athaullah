#!/bin/bash
set -e

./reset_database.sh
go build -o build/server-iot cmd/*
./build/server-iot
