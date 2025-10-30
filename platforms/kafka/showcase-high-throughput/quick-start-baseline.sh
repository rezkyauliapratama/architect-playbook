# 1. Start baseline system
docker compose up -d

# 2. Wait for ready
sleep 10

# 3. Run tests
sh run-all-baseline-tests.sh

# 4. Stop all services
docker compose down
