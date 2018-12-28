#!/bin/sh

set -e
go build

./halite --replay-directory replays/ -vvv --width 32 --height 32 "go run MyBot.go" "go run MyBot.go"
