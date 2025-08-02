#!/bin/bash

set -e

ENV_FILE=${1:-.env}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

if [[ ! -f "$ENV_FILE" ]]; then
    echo "Error: Environment file '$ENV_FILE' not found"
    echo "Usage: $0 [env-file]"
    echo "Example: $0 .env.development"
    exit 1
fi

echo "Loading environment from: $ENV_FILE"
set -a
source "$ENV_FILE"
set +a

echo "Generating Prometheus configuration..."
envsubst < "$PROJECT_ROOT/infrastructure/prometheus/prometheus.yml.template" > "$PROJECT_ROOT/infrastructure/prometheus/prometheus.yml"

echo "Checking if we need to generate Grafana datasources..."
GRAFANA_DATASOURCES_TEMPLATE="$PROJECT_ROOT/infrastructure/grafana/datasources.yml.template"
GRAFANA_DATASOURCES="$PROJECT_ROOT/infrastructure/grafana/datasources.yml"

if [[ -f "$GRAFANA_DATASOURCES_TEMPLATE" ]]; then
    echo "Generating Grafana datasources configuration..."
    envsubst < "$GRAFANA_DATASOURCES_TEMPLATE" > "$GRAFANA_DATASOURCES"
fi

echo "Configuration files generated successfully!"
echo "- Prometheus config: $PROJECT_ROOT/infrastructure/prometheus/prometheus.yml"
if [[ -f "$GRAFANA_DATASOURCES" ]]; then
    echo "- Grafana datasources: $GRAFANA_DATASOURCES"
fi

echo ""
echo "To start the services with generated configs:"
echo "docker-compose --env-file $ENV_FILE up -d"
