#!/bin/bash
# TODO: Adapt this script to your GraphDB repository configuration.
# This script loads init.ttl into the GraphDB repository.
# Requires: GRAPHDB_URL (e.g., http://graphdb:7200/repositories/rag)

set -e

REPO_URL="${GRAPHDB_URL:-http://graphdb:7200/repositories/rag}"

echo "Loading init.ttl into ${REPO_URL}..."

curl -s -X POST \
  -H "Content-Type: application/x-turtle" \
  --data-binary @init.ttl \
  "${REPO_URL}/statements"

echo " Done."
