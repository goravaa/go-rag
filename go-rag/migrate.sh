#!/bin/bash
# ------------------------------------------------------------
# Dev-only Ent + Atlas migration script using .env.dev
# ------------------------------------------------------------
set -euo pipefail

# --- Load environment variables ------------------------------
if [ ! -f ".env.dev" ]; then
  echo "[error] .env.dev not found."
  exit 1
fi

echo "[env] Loading variables from .env.dev"
# shellcheck disable=SC2046
export $(grep -v '^#' .env.dev | xargs)

# --- Config ---------------------------------------------------
MIGRATIONS_DIR="file://migrations"
ENT_SCHEMA_PATH="ent://./ent/schema"
REVISIONS_SCHEMA="atlas_schema_revisions"

# Build URLs from .env.dev
DEV_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_DEV_NAME}?sslmode=${DB_SSLMODE}&search_path=${DB_SEARCH_PATH}"
MAIN_DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}&search_path=${DB_SEARCH_PATH}"

# --- Utility --------------------------------------------------
ensure_atlas() {
  if ! command -v atlas &> /dev/null; then
    echo "[setup] Installing Atlas CLI..."
    curl -sSf https://atlasgo.sh | sh
    echo "[setup] Atlas installed."
  else
    echo "[check] Atlas CLI found."
  fi
}

usage() {
  echo "Usage: $0 <migration_name>"
  exit 1
}

# --- Validate arg ---------------------------------------------
if [ $# -lt 1 ]; then
  echo "[error] No migration name provided."
  usage
fi
MIGRATION_NAME=$1

# --- Run workflow ---------------------------------------------
echo "------------------------------------------------------------"
echo "[run] Starting migration for: $MIGRATION_NAME"
echo "------------------------------------------------------------"

ensure_atlas

echo "[step 1/3] Generating Ent code..."
go generate ./...
echo "[ok] Ent code generated."

echo "[step 2/3] Creating migration diff (using $DEV_DB_URL)..."
atlas migrate diff "$MIGRATION_NAME" \
  --dir "$MIGRATIONS_DIR" \
  --to "$ENT_SCHEMA_PATH" \
  --dev-url "$DEV_DB_URL"
echo "[ok] Migration diff created."

echo "[step 3/3] Applying migrations to $DB_NAME..."
if atlas migrate status --dir "$MIGRATIONS_DIR" --url "$MAIN_DB_URL" --revisions-schema="$REVISIONS_SCHEMA" 2>&1 | grep -q "Error:"; then
  echo "[init] No revision table found, initializing..."
  atlas migrate apply \
    --dir "$MIGRATIONS_DIR" \
    --url "$MAIN_DB_URL" \
    --revisions-schema="$REVISIONS_SCHEMA" \
    --baseline 0
else
  atlas migrate apply \
    --dir "$MIGRATIONS_DIR" \
    --url "$MAIN_DB_URL" \
    --revisions-schema="$REVISIONS_SCHEMA"\
    --allow-dirty 
fi

echo "[ok] Migrations applied."
echo "[done] Migration complete."
