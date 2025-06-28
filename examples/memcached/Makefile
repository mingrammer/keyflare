.PHONY: cluster-up cluster-down cluster-status cluster-logs run-example

# Start Memcached instances
cluster-up:
	@echo "Starting Memcached instances..."
	docker-compose up -d
	@echo "Waiting for instances to be ready..."
	sleep 10
	@echo "Memcached instances are ready!"
	@echo "Available instances:"
	@echo "- localhost:11211"
	@echo "- localhost:11212" 
	@echo "- localhost:11213"

# Stop Memcached instances
cluster-down:
	@echo "Stopping Memcached instances..."
	docker-compose down
	docker system prune -f

# Check instances status
cluster-status:
	@echo "Checking Memcached instances status..."
	@echo "Node 1 (localhost:11211):"
	@docker exec kf-memcached-node-1 sh -c "echo 'stats' | nc localhost 11211" | head -5 || echo "Node 1 not responding"
	@echo ""
	@echo "Node 2 (localhost:11212):"
	@docker exec kf-memcached-node-2 sh -c "echo 'stats' | nc localhost 11211" | head -5 || echo "Node 2 not responding"
	@echo ""
	@echo "Node 3 (localhost:11213):"
	@docker exec kf-memcached-node-3 sh -c "echo 'stats' | nc localhost 11211" | head -5 || echo "Node 3 not responding"

# View instances logs
cluster-logs:
	docker-compose logs -f

# Run KeyFlare Memcached example
run-example:
	@echo "Running KeyFlare Memcached example..."
	go run *.go

# Full demo - start instances and run example
demo: cluster-up
	@echo "Running KeyFlare demo..."
	go run *.go