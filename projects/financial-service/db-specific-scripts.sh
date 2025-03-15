#!/bin/bash
# db-specific-scripts.sh

set -e
set -u

# Wait for databases to be created
sleep 5

# Execute SQL scripts for db1
if [ -f "/docker-entrypoint-initdb.d/init-db-fund-transfer.sql" ]; then
  echo "Initializing db1 with SQL script"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "fund_transfer" -f /docker-entrypoint-initdb.d/init-db-fund-transfer.sql
fi
# Add more database-specific scripts as needed
