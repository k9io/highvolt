#!/bin/bash

export JSONAIR_PAT="<your pat>"
export JSONAIR_URL="https://key9.dev:9191"
export JSONAIR_TYPE="highvolt"
export JSONAIR_NAME="suricata.config"

export HIGHVOLT_PAT="<highvolt pat>"
export HIGHVOLT_URL="http://127.0.0.1:8181"

export SLEEP=30

#./hv-suricata
./suricata
