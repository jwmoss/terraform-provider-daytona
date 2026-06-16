#!/usr/bin/env bash
#
# daytona-access-token.sh — mint a fresh Daytona access token (JWT) for local
# acceptance testing of the organization-API resources.
#
# The organization-API resources (daytona_organization, daytona_api_key,
# daytona_organization_role, daytona_organization_invitation,
# daytona_organization_otel_config) require an access token, not an API key. Access
# tokens expire after 24h; a refresh token obtained once via `daytona login` mints
# fresh ones without a browser.
#
# This is a local developer convenience for running the JWT acceptance tests. It is
# not used by CI (CI relies on the mocked unit tests plus the API-key acceptance
# suite).
#
# The Auth0 client used for the refresh grant is the one shipped in the Daytona CLI.
# Rather than hard-code its credentials, this script reads them from your locally
# installed `daytona` binary (which contains them), or from the environment. Nothing
# secret is stored in this repository.
#
# Usage:
#   daytona login                                              # once
#   export DAYTONA_ACCESS_TOKEN="$(ci/daytona-access-token.sh)"
#   export DAYTONA_ORGANIZATION_ID=...
#   TF_ACC=1 go test -run TestAcc ./internal/provider/
#
# Overrides (all optional): DAYTONA_REFRESH_TOKEN, DAYTONA_AUTH0_DOMAIN,
# DAYTONA_AUTH0_CLIENT_ID, DAYTONA_AUTH0_CLIENT_SECRET, DAYTONA_CONFIG_PATH.

set -euo pipefail

err() {
  echo "daytona-access-token: $*" >&2
  exit 1
}

command -v curl >/dev/null || err "curl is required"
command -v jq >/dev/null || err "jq is required"

# Pull an Auth0 setting baked into the daytona CLI binary, e.g. Auth0ClientSecret.
from_cli_binary() {
  local key="$1" bin
  command -v strings >/dev/null || return 1
  bin="$(command -v daytona)" || return 1
  strings "${bin}" 2>/dev/null | grep -oE "${key}=[^']+" | head -1 | cut -d= -f2-
}

AUTH0_DOMAIN="${DAYTONA_AUTH0_DOMAIN:-$(from_cli_binary Auth0Domain || true)}"
AUTH0_DOMAIN="${AUTH0_DOMAIN:-https://daytonaio.us.auth0.com/}"
CLIENT_ID="${DAYTONA_AUTH0_CLIENT_ID:-$(from_cli_binary Auth0ClientId || true)}"
CLIENT_SECRET="${DAYTONA_AUTH0_CLIENT_SECRET:-$(from_cli_binary Auth0ClientSecret || true)}"

[[ -n "${CLIENT_ID}" ]] || err "could not determine Auth0 client id; set DAYTONA_AUTH0_CLIENT_ID"
[[ -n "${CLIENT_SECRET}" ]] || err "could not read Auth0 client secret from the daytona CLI; install daytona or set DAYTONA_AUTH0_CLIENT_SECRET"

find_config() {
  local candidates=(
    "${DAYTONA_CONFIG_PATH:-}"
    "${HOME}/Library/Application Support/daytona/config.json"
    "${XDG_CONFIG_HOME:-${HOME}/.config}/daytona/config.json"
  )
  local path
  for path in "${candidates[@]}"; do
    [[ -n "${path}" && -f "${path}" ]] && {
      printf '%s' "${path}"
      return 0
    }
  done
  return 1
}

refresh_token="${DAYTONA_REFRESH_TOKEN:-}"
if [[ -z "${refresh_token}" ]]; then
  config_path="$(find_config)" || err "set DAYTONA_REFRESH_TOKEN or run 'daytona login' (no CLI config found)"
  refresh_token="$(jq -r '.profiles[0].api.token.refreshToken // empty' "${config_path}")"
  [[ -n "${refresh_token}" ]] || err "no refresh token in ${config_path}; run 'daytona login'"
fi

response="$(
  curl -fsS --max-time 30 -X POST "${AUTH0_DOMAIN%/}/oauth/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "grant_type=refresh_token" \
    --data-urlencode "client_id=${CLIENT_ID}" \
    --data-urlencode "client_secret=${CLIENT_SECRET}" \
    --data-urlencode "refresh_token=${refresh_token}"
)" || err "token request failed (refresh token expired or revoked? re-run 'daytona login')"

access_token="$(jq -r '.access_token // empty' <<<"${response}")"
[[ -n "${access_token}" ]] || err "no access_token in response: ${response}"

printf '%s\n' "${access_token}"
