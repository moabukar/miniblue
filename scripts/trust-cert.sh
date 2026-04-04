#!/usr/bin/env bash
# Trust the miniblue self-signed certificate
# Run after starting miniblue at least once

set -e

CERT_PATH="${HOME}/.miniblue/cert.pem"

if [ ! -f "$CERT_PATH" ]; then
    echo "Certificate not found at $CERT_PATH"
    echo "Start miniblue first: ./bin/miniblue"
    exit 1
fi

echo "Certificate: $CERT_PATH"

case "$(uname -s)" in
    Darwin)
        echo "Adding to macOS system keychain (requires sudo)..."
        sudo security add-trusted-cert -d -r trustRoot \
            -k /Library/Keychains/System.keychain "$CERT_PATH"
        echo "Done. Certificate trusted system-wide."
        ;;
    Linux)
        if command -v update-ca-certificates &>/dev/null; then
            echo "Adding to system CA store (Debian/Ubuntu)..."
            sudo cp "$CERT_PATH" /usr/local/share/ca-certificates/miniblue.crt
            sudo update-ca-certificates
        elif command -v update-ca-trust &>/dev/null; then
            echo "Adding to system CA store (RHEL/Fedora)..."
            sudo cp "$CERT_PATH" /etc/pki/ca-trust/source/anchors/miniblue.crt
            sudo update-ca-trust
        else
            echo "Could not detect CA trust utility."
            echo "Manually add: $CERT_PATH to your system CA store"
            echo "Or: export SSL_CERT_FILE=$CERT_PATH"
        fi
        echo "Done."
        ;;
    *)
        echo "Unsupported OS. Add $CERT_PATH to your system trust store manually."
        echo "Or: export SSL_CERT_FILE=$CERT_PATH"
        ;;
esac
