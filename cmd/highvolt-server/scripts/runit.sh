#!/bin/bash

# Example launcher for highvolt-server. Alternatively, copy .env.example to
# .env in the working directory and run ./highvolt-server directly.
# See .env.example for a description of each variable.

# --- JSONAir (where highvolt-server fetches its configuration) ---
export JSONAIR_PAT="<your-jsonair-pat>"
export JSONAIR_URL="http://localhost:9191"
export JSONAIR_TYPE="highvolt"
export JSONAIR_NAME="highvolt-server.config"

# --- Client authentication (clients -> highvolt-server) ---
export HIGHVOLT_PAT="<pat-clients-present-to-highvolt>"
export JWT_TOKEN_SECRET="<long-random-secret>"
export JWT_TOKEN_EXPIRE=60

# --- Runtime ---
export RUNAS="highvolt"
#export RELOAD_SLEEP=60   # optional, defaults to 60

./highvolt-server
