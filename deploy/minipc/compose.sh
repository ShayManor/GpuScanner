#!/bin/sh
# docker compose with the secrets file, from any directory.
exec docker compose --env-file "$HOME/.config/gpuscanner/env" \
	-f "$(dirname "$(readlink -f "$0")")/docker-compose.yml" "$@"
