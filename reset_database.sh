#!/bin/bash

# Connection details
CONN_STRING="postgres://postgres:password@localhost"
DB_NAME="rustdemo"
INIT_FILE="./init.sql"

# Run the commands
echo "Connecting to PostgreSQL and resetting database..."

# Forcefully terminate all connections to the database
psql -q "$CONN_STRING" <<EOF
-- Terminate all connections to the database
SELECT pg_terminate_backend(pg_stat_activity.pid)
FROM pg_stat_activity
WHERE pg_stat_activity.datname = '$DB_NAME'
  AND pid <> pg_backend_pid();
EOF

# Drop and recreate the database
psql -q "$CONN_STRING" <<EOF
-- Drop the database if it exists
DROP DATABASE IF EXISTS $DB_NAME;

-- Create a new database
CREATE DATABASE $DB_NAME;
EOF

# Run the init.sql file
if [ -f "$INIT_FILE" ]; then
    echo "Executing init.sql..."
    psql -q "$CONN_STRING/$DB_NAME" -f "$INIT_FILE"
    echo "Database initialization complete."
else
    echo "Error: init.sql file not found at $INIT_FILE"
    exit 1
fi
