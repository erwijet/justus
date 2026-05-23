#!/bin/sh
set -e

atlas migrate apply --dir "file://migrations" --url "$DATABASE_URL"
exec /bin/justus
