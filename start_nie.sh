#!/bin/bash
export NIE_DATABASE_URL="postgres://nie:nie@127.0.0.1:5432/nie?sslmode=disable"
export NIE_ADDR="0.0.0.0:8080"
export NIE_DATA_DIR="./data"
export NIE_WORKER="true"
export NIE_WORKERS="2"
export NIE_MWMBL_URL="https://api.mwmbl.org"
export NIE_WAYBACK_ENABLED="true"
export NIE_SOFTWARE_HERITAGE_ENABLED="true"
export NIE_MENTIONS_DISCOVERY_ENABLED="true"
export NIE_LOG_LEVEL="info"

chmod +x ./bin/nie-linux
exec ./bin/nie-linux
