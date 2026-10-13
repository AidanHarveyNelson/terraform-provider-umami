#!/usr/bin/env bash
# Prints a .env for running acceptance tests against the docker-compose Umami
# as a non-admin user with an API key: scripts/testacc-env.sh > .env
set -euo pipefail

endpoint="${UMAMI_API_CLIENT_ENDPOINT:-http://localhost:3000/api}"
user="tfacc"
password="tfacc-password-1"

for _ in $(seq 1 60); do
  curl -fs "$endpoint/heartbeat" >/dev/null && break
  sleep 2
done

login() {
  curl -fs -X POST "$endpoint/auth/login" -H 'Content-Type: application/json' \
    -d "{\"username\":\"$1\",\"password\":\"$2\"}" | jq -r .token
}

admin_token="$(login admin umami)"
curl -s -X POST "$endpoint/users" -H "Authorization: Bearer $admin_token" -H 'Content-Type: application/json' \
  -d "{\"username\":\"$user\",\"password\":\"$password\",\"role\":\"user\"}" >/dev/null

user_token="$(login "$user" "$password")"
api_key="$(curl -fs -X POST "$endpoint/me/api-keys" -H "Authorization: Bearer $user_token" \
  -H 'Content-Type: application/json' -d '{"name":"tfacc"}' | jq -r .key)"

echo "UMAMI_API_CLIENT_ENDPOINT=$endpoint"
echo "UMAMI_API_KEY=$api_key"
