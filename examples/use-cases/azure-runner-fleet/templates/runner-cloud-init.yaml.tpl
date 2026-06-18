#cloud-config
package_update: ${package_update}
package_upgrade: ${package_upgrade}

packages:
  - ca-certificates
  - curl

write_files:
  - path: /etc/daytona/runner.env
    permissions: '0600'
    content: |
      DAYTONA_API_URL=${api_url}
      DAYTONA_RUNNER_TOKEN=${api_key}
      DAYTONA_RUNNER_ID=${runner_id}
      DAYTONA_RUNNER_NAME=${runner_name}
      DAYTONA_RUNNER_REGION=${runner_region}
      DAYTONA_RUNNER_POLL_TIMEOUT=${poll_timeout}
      DAYTONA_RUNNER_POLL_LIMIT=${poll_limit}

runcmd:
  - mkdir -p /etc/systemd/system/daytona-runner.service.d
  - curl -fsSL -o /tmp/daytona-runner.deb "https://github.com/daytonaio/daytona/releases/download/v${runner_version}/daytona-runner_${runner_version}_amd64.deb"
  - DEBIAN_FRONTEND=noninteractive apt-get install -y /tmp/daytona-runner.deb
  - |
      cat > /etc/systemd/system/daytona-runner.service.d/10-terraform-env.conf <<'EOF'
      [Service]
      EnvironmentFile=/etc/daytona/runner.env
      EOF
  - systemctl daemon-reload
  - systemctl enable --now daytona-runner
  - rm -f /tmp/daytona-runner.deb
  - systemctl status daytona-runner --no-pager || true

final_message: "Daytona runner ${runner_name} installation completed after $UPTIME seconds"
