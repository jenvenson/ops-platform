#!/bin/sh
set -eu

mysql_client_bin() {
  if command -v mariadb >/dev/null 2>&1; then
    printf '%s\n' "mariadb"
    return 0
  fi
  printf '%s\n' "mysql"
}

install_required_packages() {
  if which nmap >/dev/null 2>&1; then
    return 0
  fi

  echo "Installing required scan dependencies..."
  PACKAGES="bash nmap nmap-nselibs nmap-scripts unzip curl libc6-compat mariadb-client"

  if apk add --no-cache $PACKAGES; then
    which nmap >/dev/null 2>&1
    return 0
  fi

  echo "Retrying apk install with mirror fallback..."
  ALPINE_VERSION="$(cut -d. -f1,2 /etc/alpine-release)"
  cat >/etc/apk/repositories <<EOF
https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/main
https://mirrors.aliyun.com/alpine/v${ALPINE_VERSION}/community
EOF

  for i in 1 2 3; do
    if apk add --no-cache $PACKAGES; then
      which nmap >/dev/null 2>&1
      return 0
    fi
    sleep 3
  done

  echo "Error: failed to install required scan dependencies"
  return 1
}

install_rustscan() {
  if which rustscan >/dev/null 2>&1; then
    rustscan --version
    return 0
  fi

  echo "Installing RustScan..."
  cd /tmp
  rm -f x86_64-linux-rustscan.tar.gz.zip rustscan
  curl -fL --retry 1 --max-time 20 -O https://github.com/bee-san/RustScan/releases/download/2.4.1/x86_64-linux-rustscan.tar.gz.zip
  unzip -o x86_64-linux-rustscan.tar.gz.zip
  tar xzf x86_64-linux-rustscan.tar.gz
  mv rustscan /usr/local/bin/
  chmod +x /usr/local/bin/rustscan
  rm -rf /tmp/x86_64-* /tmp/rustscan /tmp/x86_64-linux-rustscan.tar.gz.zip
  rustscan --version
}

install_nuclei() {
  NUCLEI_VERSION="${NUCLEI_VERSION:-v3.7.1}"
  NUCLEI_ZIP="nuclei_${NUCLEI_VERSION#v}_linux_amd64.zip"
  NUCLEI_URL="${NUCLEI_BINARY_URL:-https://github.com/projectdiscovery/nuclei/releases/download/${NUCLEI_VERSION}/${NUCLEI_ZIP}}"
  LOCAL_NUCLEI_BIN="/app/backend/bin/nuclei"

  if which nuclei >/dev/null 2>&1; then
    VERSION_OUTPUT="$(nuclei -version 2>/dev/null || nuclei --version 2>/dev/null || true)"
    echo "$VERSION_OUTPUT"
    if printf '%s' "$VERSION_OUTPUT" | grep -q "Current Version: ${NUCLEI_VERSION#v}"; then
      return 0
    fi
    echo "Existing nuclei version is too old, reinstalling..."
    rm -f /usr/local/bin/nuclei
  fi

  if [ -x "$LOCAL_NUCLEI_BIN" ]; then
    echo "Installing nuclei from local mounted binary..."
    mkdir -p /usr/local/bin
    cp "$LOCAL_NUCLEI_BIN" /usr/local/bin/nuclei
    chmod +x /usr/local/bin/nuclei
    nuclei -version 2>/dev/null || nuclei --version 2>/dev/null || true
    return 0
  fi

  echo "Installing nuclei..."
  cd /tmp
  rm -f "/tmp/${NUCLEI_ZIP}" /tmp/nuclei
  if curl -fL --retry 2 --max-time 120 -o "/tmp/${NUCLEI_ZIP}" "$NUCLEI_URL" && unzip -o "/tmp/${NUCLEI_ZIP}" nuclei -d /tmp; then
    mkdir -p /usr/local/bin
    cp /tmp/nuclei /usr/local/bin/nuclei
    chmod +x /usr/local/bin/nuclei
    nuclei -version 2>/dev/null || nuclei --version 2>/dev/null || true
    rm -f "/tmp/${NUCLEI_ZIP}" /tmp/nuclei
    return 0
  fi

  echo "Warning: nuclei binary download failed, attempting go install fallback"
  if go install github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest; then
    GOPATH_BIN="$(go env GOPATH)/bin/nuclei"
    mkdir -p /usr/local/bin
    cp "$GOPATH_BIN" /usr/local/bin/nuclei
    chmod +x /usr/local/bin/nuclei
    nuclei -version 2>/dev/null || nuclei --version 2>/dev/null || true
    return 0
  fi

  echo "Warning: nuclei install failed, continuing without nuclei"
  return 0
}

init_database_schema() {
  if [ ! -f /app/backend/scripts/init.sql ]; then
    return 0
  fi

  echo "Initializing database schema..."
  MYSQL_PWD="${DB_PASSWORD}" "$(mysql_client_bin)" --skip-ssl --protocol=TCP -h "${DB_HOST:-mysql}" -P "${DB_PORT:-3306}" -u "${DB_USER:-root}" < /app/backend/scripts/init.sql
}

apply_database_migrations() {
  if [ ! -f /app/deploy/apply-migrations.sh ]; then
    return 0
  fi

  echo "Applying database migrations..."
  DB_HOST="${DB_HOST:-mysql}" \
  DB_PORT="${DB_PORT:-3306}" \
  DB_USER="${DB_USER:-root}" \
  DB_PASSWORD="${DB_PASSWORD}" \
  DB_NAME="${DB_NAME:-ops_platform}" \
  bash /app/deploy/apply-migrations.sh --direct
}

if [ -f /opt/.dev-base-image ]; then
  echo "Dev base image detected, skipping tool installation."
else
  install_required_packages
  install_rustscan || echo "Warning: RustScan install failed, continuing without rustscan"
  install_nuclei
fi
init_database_schema
apply_database_migrations

echo "Starting backend..."
cd /app/backend
exec go run ./cmd/server/main.go
