#!/bin/bash

export JSONAIR_PAT="<your pat>"
export JSONAIR_URL="https://key9.dev:9191"
export JSONAIR_TYPE="highvolt"
export JSONAIR_NAME="suricata.config"

export SLEEP=30

#./hv-suricata
./suricata
