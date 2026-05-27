#!/bin/bash

export JSONAIR_UUID="JSONAIR_UUID"
export JSONAIR_KEY="JSONAIR_KEY"
export JSONAIR_URL="http://localhost:9191"
export JSONAIR_TYPE="highvolt"
export JSONAIR_NAME="highvolt-server.config"

export MAX_QUEUE_SIZE=100000000

./highvolt-server
