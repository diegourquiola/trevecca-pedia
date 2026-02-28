#!/bin/bash
# setup-db.sh - Apply auth schema to fly.io Postgres
# NOTE: Run `fly postgres attach` first so the database exists.

set -e

DB_APP_NAME="${1:-trevecca-pedia-db}"
DB_NAME="${2:-trevecca_pedia_auth}"

echo "========================================="
echo "Auth DB Schema Setup"
echo "Database app: $DB_APP_NAME"
echo "Target database: $DB_NAME"
echo "========================================="
echo ""

if ! command -v fly &> /dev/null; then
    echo "Error: fly CLI is not installed"
    echo "Install it from: https://fly.io/docs/hands-on/install-flyctl/"
    exit 1
fi

if ! fly auth whoami &> /dev/null; then
    echo "Error: Not logged into fly.io"
    echo "Run: fly auth login"
    exit 1
fi

if ! fly status --app "$DB_APP_NAME" &> /dev/null; then
    echo "Error: Database app '$DB_APP_NAME' not found"
    echo "Create it first: fly postgres create --name $DB_APP_NAME"
    exit 1
fi

echo "Applying schema files..."

for file in init/0001_init.sql init/0002_whitelist.sql; do
    if [ -f "$file" ]; then
        echo "  Applying $file..."
        printf '%s\n\\q\n' "$(cat "$file")" | fly postgres connect --app "$DB_APP_NAME" --database "$DB_NAME"
        echo "  ✓ Applied $file"
    else
        echo "  Warning: $file not found, skipping"
    fi
done

echo ""
echo "========================================="
echo "Schema applied successfully!"
echo "========================================="
