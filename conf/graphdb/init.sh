#!/bin/bash

REPO_ID=${REPOSITORY_ID:-myrepo}
GRAPHDB_URL="http://${GRAPHDB_HOST:-localhost}:${GRAPHDB_PORT:-7200}"

# Wait for GraphDB to be ready
echo "Waiting for GraphDB to be ready..."
until curl -s -f "$GRAPHDB_URL/rest/repositories" > /dev/null; do
    echo "GraphDB not ready yet, waiting..."
    sleep 5
done

# Check if repository already exists
if curl -s -f "$GRAPHDB_URL/rest/repositories/$REPO_ID" > /dev/null; then
    echo "Repository '$REPO_ID' already exists, skipping creation."
else
    echo "Creating repository '$REPO_ID'..."
    curl -X POST \
        "$GRAPHDB_URL/rest/repositories" \
        -H 'Content-Type: multipart/form-data' \
        -F "config=@/repository.ttl"

    if [ $? -eq 0 ]; then
        echo "Repository '$REPO_ID' created successfully."
    else
        echo "Failed to create repository."
        exit 1
    fi
fi

# Load data if file exists and is not empty
if [ -s /data.ttl ]; then
    echo "Loading data from /data.ttl into repository '$REPO_ID'..."
    curl -X POST \
        "$GRAPHDB_URL/repositories/$REPO_ID/statements" \
        -H 'Content-Type: text/turtle' \
        -T /data.ttl

    if [ $? -eq 0 ]; then
        echo "Data loaded successfully."
    else
        echo "Failed to load data."
        exit 1
    fi
else
    echo "No data file provided or file is empty. Skipping data loading."
fi

echo "Initialization complete"
