#!/bin/bash

CONTAINER_NAME="hn-postgres"
IMAGE_NAME="hn-postgres:latest"
DB_PORT=5432
DB_NAME="scraperdb"
DB_USER="scraperuser"
DB_PASSWORD="supersecret"
DOCKER_DIR="postgres"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

start_db() {
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            echo -e "${YELLOW}Database is already running${NC}"
        else
            echo "Starting existing container..."
            docker start "$CONTAINER_NAME"
            echo -e "${GREEN}✓${NC} Database started"
        fi
    else
        if ! docker images --format '{{.Repository}}:{{.Tag}}' | grep -q "^${IMAGE_NAME}$"; then
            echo "Building Docker image..."
            cd "$DOCKER_DIR"
            docker build -t "$IMAGE_NAME" .
            cd - > /dev/null
            echo -e "${GREEN}✓${NC} Image built"
        fi
        
        echo "Creating new container..."
        docker run -d \
            --name "$CONTAINER_NAME" \
            -p "${DB_PORT}:5432" \
            -v "${CONTAINER_NAME}-data:/var/lib/postgresql/data" \
            --restart unless-stopped \
            "$IMAGE_NAME"
        
        echo -e "${GREEN}✓${NC} Database container created and started"
        
        echo -n "Waiting for database..."
        for i in {1..30}; do
            if docker exec "$CONTAINER_NAME" pg_isready -U "$DB_USER" -d "$DB_NAME" &> /dev/null; then
                echo -e " ${GREEN}Ready!${NC}"
                echo ""
                echo "Connection: postgres://${DB_USER}:${DB_PASSWORD}@localhost:${DB_PORT}/${DB_NAME}"
                break
            fi
            echo -n "."
            sleep 1
        done
    fi
}

stop_db() {
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo "Stopping database..."
        docker stop "$CONTAINER_NAME"
        echo -e "${GREEN}✓${NC} Database stopped"
    else
        echo -e "${YELLOW}Database is not running${NC}"
    fi
}

restart_db() {
    echo "Restarting database..."
    stop_db
    sleep 2
    start_db
}

remove_db() {
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker stop "$CONTAINER_NAME"
    fi
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo "Removing container (keeping data volume)..."
        docker rm "$CONTAINER_NAME"
        echo -e "${GREEN}✓${NC} Container removed (data preserved)"
    else
        echo -e "${YELLOW}No container to remove${NC}"
    fi
}

clean_db() {
    echo -e "${YELLOW}WARNING: This will delete all data!${NC}"
    read -p "Are you sure? (yes/no): " confirm
    
    if [ "$confirm" = "yes" ]; then
        if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            docker stop "$CONTAINER_NAME"
        fi
        
        if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            docker rm "$CONTAINER_NAME"
        fi
        
        if docker volume ls --format '{{.Name}}' | grep -q "^${CONTAINER_NAME}-data$"; then
            docker volume rm "${CONTAINER_NAME}-data"
            echo -e "${GREEN}✓${NC} Data volume removed"
        fi
        
        echo -e "${GREEN}✓${NC} Cleanup complete"
    else
        echo "Cleanup cancelled"
    fi
}

status_db() {
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo -e "${GREEN}✓${NC} Database is running"
        
        echo ""
        docker ps --filter "name=${CONTAINER_NAME}" --format "table {{.Status}}\t{{.Ports}}"
        
        if docker exec "$CONTAINER_NAME" pg_isready -U "$DB_USER" -d "$DB_NAME" &> /dev/null; then
            echo ""
            echo -n "Posts in database: "
            docker exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM posts;" 2>/dev/null | tr -d ' ' || echo "0"
        fi
    else
        if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
            echo -e "${YELLOW}Database container exists but is stopped${NC}"
            echo "Start with: $0 --startdb"
        else
            echo -e "${RED}✗${NC} Database container does not exist"
            echo "Create with: $0 --startdb"
        fi
    fi
}

connect_db() {
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker exec -it "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME"
    else
        echo -e "${RED}Database is not running${NC}"
        echo "Start with: $0 --startdb"
    fi
}

logs_db() {
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker logs "$CONTAINER_NAME" --tail 50 -f
    else
        echo -e "${RED}Container does not exist${NC}"
    fi
}

exec_sql() {
    if [ -z "$1" ]; then
        echo "Usage: $0 --exec \"SQL COMMAND\""
        return
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -c "$1"
    else
        echo -e "${RED}Database is not running${NC}"
    fi
}

show_help() {
    cat << EOF
Database Management for HN Scraper

Usage: $0 [OPTION]

Options:
  --startdb       Start the database container
  --stopdb        Stop the database container
  --restartdb     Restart the database container
  --statusdb      Check database status
  --removedb      Remove container (keep data)
  --cleandb       Remove container AND data
  --connectdb     Connect to psql console
  --logsdb        Show container logs
  --exec "SQL"    Execute SQL command

Quick Commands:
  $0 --startdb    # Start database
  $0 --stopdb     # Stop database
  $0 --statusdb   # Check status

EOF
}


case "${1}" in
    --startdb)
        start_db
        ;;
    --stopdb)
        stop_db
        ;;
    --restartdb)
        restart_db
        ;;
    --statusdb)
        status_db
        ;;
    --removedb)
        remove_db
        ;;
    --cleandb)
        clean_db
        ;;
    --connectdb)
        connect_db
        ;;
    --logsdb)
        logs_db
        ;;
    --exec)
        exec_sql "$2"
        ;;
    --help|-h|"")
        show_help
        ;;
    *)
        echo -e "${RED}Unknown option: $1${NC}"
        echo "Use --help for usage"
        exit 1
        ;;
esac