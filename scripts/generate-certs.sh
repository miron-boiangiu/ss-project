#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SECRETS_DIR="${ROOT_DIR}/secrets"
mkdir -p "${SECRETS_DIR}"

DAYS=${DAYS:-3650}
BITS=${BITS:-2048}

echo "Generating CA key and certificate..."
openssl genrsa -out "${SECRETS_DIR}/ca.key" "${BITS}"
openssl req -x509 -new -nodes -key "${SECRETS_DIR}/ca.key" \
  -sha256 -days "${DAYS}" \
  -out "${SECRETS_DIR}/ca.crt" \
  -subj "/C=RO/O=SSProject/CN=SSProject CA"

gen_cert() {
  local name=$1
  local cn=$2
  echo "Generating certificate for '${name}' (CN=${cn})..."
  openssl genrsa -out "${SECRETS_DIR}/${name}.key" "${BITS}"
  openssl req -new -key "${SECRETS_DIR}/${name}.key" \
    -out "${SECRETS_DIR}/${name}.csr" \
    -subj "/C=RO/O=SSProject/CN=${cn}"
  cat > "${SECRETS_DIR}/${name}.ext" <<EOF
subjectAltName=DNS:${cn}
EOF
  openssl x509 -req -in "${SECRETS_DIR}/${name}.csr" \
    -CA "${SECRETS_DIR}/ca.crt" -CAkey "${SECRETS_DIR}/ca.key" \
    -CAcreateserial -out "${SECRETS_DIR}/${name}.crt" \
    -days "${DAYS}" -sha256 \
    -extfile "${SECRETS_DIR}/${name}.ext"
  rm -f "${SECRETS_DIR}/${name}.csr" "${SECRETS_DIR}/${name}.ext"
}

gen_cert "server" "broker"
gen_cert "web" "web"
gen_cert "python-sender-1" "python-sender-1"

cp "${SECRETS_DIR}/python-sender-1.crt" "${SECRETS_DIR}/folder-uploader.crt"
cp "${SECRETS_DIR}/python-sender-1.key" "${SECRETS_DIR}/folder-uploader.key"

chmod 644 "${SECRETS_DIR}"/*.key "${SECRETS_DIR}/ca.key"

echo "Done! Certificates generated in ${SECRETS_DIR}"
ls -la "${SECRETS_DIR}"
