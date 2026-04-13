#!/bin/bash
# Start both backend and frontend in a single terminal.
# Usage: ./dev.sh
# Press Ctrl+C to stop everything.

set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

# --- Load .env ---
if [ -f .env ]; then
    echo "Loading .env..."
    set -a
    source .env
    set +a
fi

# --- Cleanup on exit ---
cleanup() {
    trap - EXIT INT TERM
    echo ""
    echo "Shutting down..."
    [ -n "$BACKEND_PID" ]  && kill "$BACKEND_PID"  2>/dev/null
    [ -n "$FRONTEND_PID" ] && kill "$FRONTEND_PID" 2>/dev/null
    wait 2>/dev/null
    echo "Done."
}
trap cleanup EXIT INT TERM

# --- Build & start backend ---
echo "Building backend..."
(cd apps/server && go build -o "$ROOT_DIR/apps/server/server" ./cmd/server)

echo "Starting backend on :8080..."
"$ROOT_DIR/apps/server/server" &
BACKEND_PID=$!

# --- Install deps & start frontend ---
echo "Starting frontend on :4200..."
(cd apps/frontend && npm install --silent && npm start) &
FRONTEND_PID=$!

echo ""
echo "========================================="
echo "  Backend  → http://localhost:8080"
echo "  Frontend → http://localhost:4200"
echo "  Press Ctrl+C to stop both"
echo "========================================="
echo ""

# --- Keep running until a process dies or user hits Ctrl+C ---
while kill -0 "$BACKEND_PID" 2>/dev/null && kill -0 "$FRONTEND_PID" 2>/dev/null; do
    sleep 1
done
