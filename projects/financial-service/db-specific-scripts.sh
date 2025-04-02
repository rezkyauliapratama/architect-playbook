#!/bin/bash
# db-specific-scripts.sh

set -e
set -u

# Wait for databases to be created
sleep 5

# Execute SQL scripts for db1
if [ -f "/docker-entrypoint-initdb.d/init-db-notification.sql" ]; then
  echo "Initializing db1 with SQL script"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "notification" -f /docker-entrypoint-initdb.d/init-db-notification.sql
fi

# Execute SQL scripts for db2
if [ -f "/docker-entrypoint-initdb.d/init-db-bifast.sql" ]; then
  echo "Initializing db1 with SQL script"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "bifast" -f /docker-entrypoint-initdb.d/init-db-bifast.sql
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "bifast" -f /docker-entrypoint-initdb.d/seeding-db-bifast.sql
fi

# Add more database-specific scripts as needed
