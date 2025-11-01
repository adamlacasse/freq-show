#!/bin/bash
# Load environment variables from .env file and start the server

# Change to repository root
cd "$(dirname "$0")/../.."

# Load .env file if it exists
if [ -f .env ]; then
    echo "Loading environment variables from .env..."
    set -a  # automatically export all variables
    source .env
    set +a
fi

# Start the server
echo "Starting FreqShow server..."
./apps/server/server
