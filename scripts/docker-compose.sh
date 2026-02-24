#!/bin/bash
# Helper script for managing Docker Compose services (proxy + Chrome)
# Run from project root: ./scripts/docker-compose.sh [command]

set -e

# Change to docker directory
cd "$(dirname "$0")/../docker"

COMMAND="${1:-help}"

case "$COMMAND" in
  start)
    echo "Starting Markdown in the Middle (proxy + Chrome)..."
    docker compose up -d
    echo ""
    echo "✅ Services started!"
    echo ""
    echo "Proxy available at:"
    echo "  HTTP:  http://localhost:8080"
    echo "  HTTPS: https://localhost:8080 (with --proxy-cacert ./certs/cert.pem)"
    echo ""
    echo "Chrome DevTools available at: http://localhost:9222"
    echo ""
    echo "View logs with: docker compose logs -f"
    ;;

  stop)
    echo "Stopping services..."
    docker compose down
    echo "✅ Services stopped"
    ;;

  restart)
    echo "Restarting services..."
    docker compose restart
    echo "✅ Services restarted"
    ;;

  logs)
    docker compose logs -f "$2"
    ;;

  proxy-logs)
    docker compose logs -f proxy
    ;;

  chrome-logs)
    docker compose logs -f chrome
    ;;

  status)
    echo "Service status:"
    docker compose ps
    ;;

  build)
    echo "Building Docker image..."
    docker compose build
    echo "✅ Build complete"
    ;;

  shell)
    echo "Opening shell in proxy container..."
    docker compose exec proxy sh
    ;;

  clean)
    echo "Removing all containers, volumes, and images..."
    docker compose down -v --rmi all
    echo "✅ Cleaned up"
    ;;

  test)
    echo "Testing proxy..."
    echo ""
    echo "HTTP request:"
    curl -x http://localhost:8080 -s http://example.com | head -20
    echo ""
    echo "✅ Test complete"
    ;;

  help|*)
    echo "Markdown in the Middle - Docker Compose Helper"
    echo ""
    echo "Usage: ./scripts/docker-compose.sh [command]"
    echo ""
    echo "Commands:"
    echo "  start           - Start proxy + Chrome services"
    echo "  stop            - Stop all services"
    echo "  restart         - Restart services"
    echo "  status          - Show service status"
    echo "  logs [service]  - View logs (proxy, chrome, or all)"
    echo "  proxy-logs      - View proxy logs only"
    echo "  chrome-logs     - View Chrome logs only"
    echo "  build           - Build/rebuild Docker image"
    echo "  shell           - Open shell in proxy container"
    echo "  test            - Test the proxy with a sample request"
    echo "  clean           - Remove all containers and volumes"
    echo "  help            - Show this message"
    echo ""
    echo "Examples:"
    echo "  ./scripts/docker-compose.sh start"
    echo "  ./scripts/docker-compose.sh logs proxy"
    echo "  ./scripts/docker-compose.sh test"
    ;;
esac
