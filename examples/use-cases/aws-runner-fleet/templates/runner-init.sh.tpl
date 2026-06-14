#!/bin/bash
set -euo pipefail

# Written by Terraform from the daytona_runner resource. The runner daemon
# reads these on startup to authenticate and join its region.
mkdir -p /etc/daytona
cat > /etc/daytona/runner.env <<EOF
DAYTONA_RUNNER_ID=${runner_id}
DAYTONA_RUNNER_NAME=${runner_name}
DAYTONA_RUNNER_API_KEY=${api_key}
DAYTONA_RUNNER_REGION=${region}
DAYTONA_API_URL=${api_url}
EOF
chmod 600 /etc/daytona/runner.env

# Install and start the Daytona runner daemon. Replace this with the install
# command from your Daytona runner deployment docs; it should load the env file
# written above (for example via an EnvironmentFile= line in its systemd unit).
# curl -fsSL https://download.daytona.io/runner/install.sh | bash
