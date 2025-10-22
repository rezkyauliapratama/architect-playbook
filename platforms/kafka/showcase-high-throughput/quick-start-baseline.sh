# 1. Start baseline system
docker compose -f docker-compose-baseline.yml up -d

# 2. Wait for ready
sleep 30

# 3. Create topic
sh create-topic.sh
# 4. Run tests
sh run-all-baseline-tests.sh
