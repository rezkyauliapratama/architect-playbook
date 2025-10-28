# 1. Start baseline system
docker compose up -d

# 2. Wait for ready
sleep 10

# 3. Create topic
sh create-topic.sh
# 4. Run tests
sh run-all-baseline-tests.sh

docker compose down
