.PHONY: cluster-up cluster-down cluster-status cluster-logs run-example

# Start Redis cluster
cluster-up:
	@echo "Starting Redis cluster..."
	docker-compose up -d
	@echo "Waiting for nodes to be ready..."
	sleep 10
	@echo "Testing individual node connections..."
	docker exec kf-redis-cluster-node-1 redis-cli -p 6379 ping
	docker exec kf-redis-cluster-node-2 redis-cli -p 6379 ping
	docker exec kf-redis-cluster-node-3 redis-cli -p 6379 ping
	@echo "Creating cluster using Docker host IP..."
	docker run --rm --network host redis:7.2-alpine redis-cli --cluster create host.docker.internal:7001 host.docker.internal:7002 host.docker.internal:7003 --cluster-replicas 0 --cluster-yes
	@echo "Redis cluster is ready!"
	@echo "Cluster nodes: host.docker.internal:7001, host.docker.internal:7002, host.docker.internal:7003"

# Stop Redis cluster
cluster-down:
	@echo "Stopping Redis cluster..."
	docker-compose down
	docker system prune -f

# Check cluster status
cluster-status:
	@echo "Checking Redis cluster status..."
	docker exec kf-redis-cluster-node-1 redis-cli -p 6379 cluster info
	docker exec kf-redis-cluster-node-1 redis-cli -p 6379 cluster nodes

# View cluster logs
cluster-logs:
	docker-compose logs -f

# Run KeyFlare Redis example
run-example:
	@echo "Running KeyFlare Redis example..."
	go run *.go

# Full demo - start cluster and run example
demo: cluster-up
	@echo "Waiting for cluster to be fully ready..."
	sleep 5
	@echo "Running KeyFlare demo..."
	go run *.go