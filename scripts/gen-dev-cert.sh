#!/usr/bin/env bash
# 生成商家端 dev server 自签名 HTTPS 证书（含本机所有网卡 IP 的 SAN）。
# 手机通过 https://<局域网IP>:5174 访问时，浏览器需手动信任一次该证书，
# 之后摄像头（getUserMedia）才可用——这是浏览器安全策略，非系统 bug。
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)/certs"
mkdir -p "$DIR"
KEY="$DIR/dev-key.pem"
CERT="$DIR/dev-cert.pem"

if [ -f "$KEY" ] && [ -f "$CERT" ]; then
  # 已存在且未过期（剩余 >30 天）则跳过
  if openssl x509 -checkend 2592000 -noout -in "$CERT" >/dev/null 2>&1; then
    echo "dev 证书已存在且有效: $CERT"
    exit 0
  fi
fi

SAN="DNS:localhost,IP:127.0.0.1"
for ip in $(hostname -I 2>/dev/null || true); do
  case "$ip" in
    *.*|*:*) SAN="$SAN,IP:$ip" ;;
  esac
done

openssl req -x509 -newkey rsa:2048 -sha256 -days 3650 -nodes \
  -keyout "$KEY" -out "$CERT" \
  -subj "/CN=anmo-admin-dev" \
  -addext "subjectAltName=$SAN" \
  -addext "keyUsage=digitalSignature,keyEncipherment" \
  -addext "extendedKeyUsage=serverAuth" >/dev/null 2>&1

echo "已生成 dev 证书: $CERT"
echo "SAN: $SAN"
