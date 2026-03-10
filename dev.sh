#!/usr/bin/env bash
set -euo pipefail

# dev.sh - Start the TreveccaPedia development environment
#
# Usage:
#   ./dev.sh [services...]    Start all services in Docker except the listed ones
#   ./dev.sh stop             Stop all Docker services
#   ./dev.sh status           Show status of all Docker services
#   ./dev.sh logs [service]   Tail logs for a service (or all services)
#
# Examples:
#   ./dev.sh wiki web         Work on wiki and web locally; everything else in Docker
#   ./dev.sh auth             Work on auth locally; everything else in Docker
#   ./dev.sh                  Run everything in Docker (no local services)
#   ./dev.sh stop             Shut down all Docker services

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

ALL_SERVICES=(wiki search auth api-layer web)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BOLD}$1${NC}"
    echo "─────────────────────────────────────────"
}

print_service() {
    local name=$1
    local mode=$2
    if [[ "$mode" == "docker" ]]; then
        echo -e "  ${GREEN}[Docker]${NC}  $name"
    else
        echo -e "  ${YELLOW}[Local]${NC}   $name  -->  cd $name && air"
    fi
}

# Handle subcommands
case "${1:-}" in
    stop)
        print_header "Stopping all Docker services"
        docker compose down
        echo -e "\n${GREEN}All services stopped.${NC}"
        exit 0
        ;;
    status)
        print_header "Service status"
        docker compose ps
        exit 0
        ;;
    logs)
        shift
        docker compose logs -f "$@"
        exit 0
        ;;
    -h|--help|help)
        echo "Usage: ./dev.sh [services...]"
        echo ""
        echo "Start the TreveccaPedia development environment."
        echo "Services listed as arguments will NOT be started in Docker,"
        echo "so you can run them locally with air for live reloading."
        echo ""
        echo "Available services: ${ALL_SERVICES[*]}"
        echo ""
        echo "Commands:"
        echo "  ./dev.sh [services...]    Start environment (listed services run locally)"
        echo "  ./dev.sh stop             Stop all Docker services"
        echo "  ./dev.sh status           Show Docker service status"
        echo "  ./dev.sh logs [service]   Tail logs (optionally for a specific service)"
        echo ""
        echo "Examples:"
        echo "  ./dev.sh wiki web         Work on wiki and web locally"
        echo "  ./dev.sh auth             Work on auth locally"
        echo "  ./dev.sh                  Run everything in Docker"
        exit 0
        ;;
esac

# Collect local services from arguments
LOCAL_SERVICES=("$@")

# Validate that all provided service names are valid
for svc in "${LOCAL_SERVICES[@]}"; do
    found=false
    for valid in "${ALL_SERVICES[@]}"; do
        if [[ "$svc" == "$valid" ]]; then
            found=true
            break
        fi
    done
    if [[ "$found" == "false" ]]; then
        echo -e "${RED}Error: Unknown service '$svc'${NC}"
        echo "Valid services: ${ALL_SERVICES[*]}"
        exit 1
    fi
done

# Determine which services go to Docker
DOCKER_SERVICES=()
for svc in "${ALL_SERVICES[@]}"; do
    is_local=false
    for local_svc in "${LOCAL_SERVICES[@]}"; do
        if [[ "$svc" == "$local_svc" ]]; then
            is_local=true
            break
        fi
    done
    if [[ "$is_local" == "false" ]]; then
        DOCKER_SERVICES+=("$svc")
    fi
done

# Build the profile flags for docker compose
PROFILE_FLAGS=()
for svc in "${DOCKER_SERVICES[@]}"; do
    PROFILE_FLAGS+=(--profile "$svc")
done

# --- Start everything ---

print_header "TreveccaPedia Dev Environment"

# Always start databases first
echo -e "\n${BLUE}Starting databases...${NC}"
docker compose up -d wiki-db auth-db

# Wait for databases to be healthy
echo -e "${BLUE}Waiting for databases to be healthy...${NC}"
docker compose exec wiki-db sh -c 'until pg_isready -U wiki_user -d wiki; do sleep 1; done' 2>/dev/null
docker compose exec auth-db sh -c 'until pg_isready -U ${POSTGRES_USER:-auth_user} -d ${POSTGRES_DB:-auth} -p 5433; do sleep 1; done' 2>/dev/null
echo -e "${GREEN}Databases are ready.${NC}"

# Start Docker services (if any)
if [[ ${#DOCKER_SERVICES[@]} -gt 0 ]]; then
    echo -e "\n${BLUE}Starting services in Docker: ${DOCKER_SERVICES[*]}${NC}"
    docker compose "${PROFILE_FLAGS[@]}" up -d --build "${DOCKER_SERVICES[@]}"
fi

# Print summary
print_header "Environment Ready"

echo -e "\n${BOLD}Databases:${NC}"
echo -e "  ${GREEN}[Docker]${NC}  wiki-db   (localhost:5432)"
echo -e "  ${GREEN}[Docker]${NC}  auth-db   (localhost:5433)"

echo -e "\n${BOLD}Services:${NC}"
for svc in "${ALL_SERVICES[@]}"; do
    is_local=false
    for local_svc in "${LOCAL_SERVICES[@]}"; do
        if [[ "$svc" == "$local_svc" ]]; then
            is_local=true
            break
        fi
    done
    if [[ "$is_local" == "true" ]]; then
        print_service "$svc" "local"
    else
        print_service "$svc" "docker"
    fi
done

if [[ ${#LOCAL_SERVICES[@]} -gt 0 ]]; then
    echo ""
    echo -e "${YELLOW}Start your local services:${NC}"
    for svc in "${LOCAL_SERVICES[@]}"; do
        echo -e "  cd ${svc} && air"
    done
fi

echo ""
echo -e "Run ${BOLD}./dev.sh stop${NC} to shut everything down."
echo -e "Run ${BOLD}./dev.sh logs [service]${NC} to view logs."
echo ""
