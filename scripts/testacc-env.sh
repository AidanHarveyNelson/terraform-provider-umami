#!/usr/bin/env bash
# Prints a .env for running acceptance tests against the docker-compose Umami
# as a non-admin user: scripts/testacc-env.sh [api_key|login|fallback] > .env
# It also creates a second user for team membership tests.
#   api_key   a real API key (Umami v3.4.0+)
#   login     username and password
#   fallback  an invalid API key plus username and password, to exercise the
#             provider falling back to login when the key is rejected
set -euo pipefail

mode="${1:-api_key}"
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
create_user() {
  local id
  id="$(curl -s -X POST "$endpoint/users" -H "Authorization: Bearer $admin_token" -H 'Content-Type: application/json' \
    -d "{\"username\":\"$1\",\"password\":\"$password\",\"role\":\"$2\"}" | jq -r '.id // empty')"
  if [ -z "$id" ]; then
    id="$(curl -fs "$endpoint/admin/users?search=$1" -H "Authorization: Bearer $admin_token" |
      jq -r ".data[] | select(.username == \"$1\") | .id")"
  fi
  echo "$id"
}

create_user "$user" user >/dev/null
member_id="$(create_user tfacc-member view-only)"

echo "UMAMI_API_CLIENT_ENDPOINT=$endpoint"
echo "UMAMI_TEST_MEMBER_USER_ID=$member_id"

case "$mode" in
  api_key)
    user_token="$(login "$user" "$password")"
    api_key="$(curl -fs -X POST "$endpoint/me/api-keys" -H "Authorization: Bearer $user_token" \
      -H 'Content-Type: application/json' -d '{"name":"tfacc"}' | jq -r .key)"
    echo "UMAMI_API_KEY=$api_key"
    ;;
  login | fallback)
    [ "$mode" = fallback ] && echo "UMAMI_API_KEY=umami_invalidinvalidinvalidinvalid00"
    echo "UMAMI_USERNAME=$user"
    echo "UMAMI_PASSWORD=$password"
    ;;
  *)
    echo "unknown mode: $mode" >&2
    exit 1
    ;;
esac
