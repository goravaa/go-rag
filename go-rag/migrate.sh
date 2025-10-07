#!/bin/bash

# A script to automate the Ent schema migration workflow using Atlas.
#
# This script ensures that any command failure will halt the entire process
# to prevent inconsistent states.
set -e

# --- Configuration ---
# Adjust these variables to match your project's setup.

# The directory where migration files are stored.
MIGRATIONS_DIR="file://migrations"

# The path to your Ent schema definition.
ENT_SCHEMA_PATH="ent://./ent/schema"

# Your development database URL, used by Atlas to safely calculate changes.
# Replace with your actual credentials.
DEV_DB_URL="postgres://raguser:ragpass@localhost:5432/atlas_dev?sslmode=disable&search_path=public"

# Your main application database URL.
# It is recommended to use environment variables for production credentials.
PROD_DB_URL="postgres://raguser:ragpass@localhost:5432/ragdb?sslmode=disable&search_path=public"

# The custom schema where Atlas stores its migration history table.
# Leave this empty if you are using the default 'public' schema.
REVISIONS_SCHEMA="atlas_schema_revisions"
# --- End of Configuration ---

# Check if a migration name was provided as an argument.
if [ -z "$1" ]; then
  echo "❌ Error: No migration name provided."
  echo "Usage: ./migrate.sh <your_migration_name>"
  exit 1
fi

MIGRATION_NAME=$1

# --- Workflow ---

echo "▶️  Starting database migration process for: $MIGRATION_NAME"
echo "-----------------------------------------------------"

# Step 1: Generate Ent code from schema.
echo "🔄 Step 1/3: Generating Ent code..."
go generate ./...
echo "✅ Ent code generated successfully."
echo ""

# Step 2: Create a new migration file with Atlas.
echo "📝 Step 2/3: Creating new migration file..."
atlas migrate diff "$MIGRATION_NAME" \
  --dir "$MIGRATIONS_DIR" \
  --to "$ENT_SCHEMA_PATH" \
  --dev-url "$DEV_DB_URL"
echo "✅ Migration file created successfully."
echo ""

# Step 3: Apply all pending migrations.
echo "🚀 Step 3/3: Applying all pending migrations..."
atlas migrate apply \
  --dir "$MIGRATIONS_DIR" \
  --url "$PROD_DB_URL" \
  --revisions-schema="$REVISIONS_SCHEMA"
echo "✅ Migrations applied successfully to the database."
echo ""

echo "🎉 Migration process complete!"