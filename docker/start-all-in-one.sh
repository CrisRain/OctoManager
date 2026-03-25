#!/usr/bin/env bash
set -euo pipefail

export API_ADDR="${API_ADDR:-:8080}"
export WEB_DIST_DIR="${WEB_DIST_DIR:-/app/web-dist}"
export PYTHON_BIN="${PYTHON_BIN:-python3}"
export PLUGINS_DIR="${PLUGINS_DIR:-/app/plugins/modules}"
export PLUGIN_SDK_DIR="${PLUGIN_SDK_DIR:-/app/plugins/sdk/python}"
export LOG_LEVEL="${LOG_LEVEL:-info}"
export DATABASE_DSN="${DATABASE_DSN:-}"

db_conn="${DATABASE_DSN}"
worker_pid=""
api_pid=""
shutdown_requested=0
worker_status=0
api_status=0

wait_for_postgres() {
  local conn="$1"
  local max_attempts="${2:-60}"
  local sleep_sec="${3:-2}"

  echo "Waiting for PostgreSQL..."
  for i in $(seq 1 "${max_attempts}"); do
    if psql "${conn}" -c "SELECT 1" >/dev/null 2>&1; then
      echo "PostgreSQL is ready."
      return 0
    fi
    if [[ "${i}" -eq "${max_attempts}" ]]; then
      echo "PostgreSQL is not reachable."
      return 1
    fi
    sleep "${sleep_sec}"
  done
}

terminate_children() {
  if [[ "${shutdown_requested}" -eq 1 ]]; then
    return
  fi

  shutdown_requested=1
  echo "Stopping services..."

  if [[ -n "${api_pid}" ]] && kill -0 "${api_pid}" 2>/dev/null; then
    kill -TERM "${api_pid}" 2>/dev/null || true
  fi
  if [[ -n "${worker_pid}" ]] && kill -0 "${worker_pid}" 2>/dev/null; then
    kill -TERM "${worker_pid}" 2>/dev/null || true
  fi
}

trap terminate_children INT TERM

if [[ -n "${db_conn}" ]]; then
  wait_for_postgres "${db_conn}"
else
  echo "DATABASE_DSN is empty; PostgreSQL checks are skipped."
fi

# In the new architecture we start API and Worker in the background
echo "Starting migrations..."
/app/migrate

echo "Starting worker..."
/app/worker &
worker_pid="$!"

echo "Starting API server..."
/app/api &
api_pid="$!"

set +e
wait -n -p exited_pid "${worker_pid}" "${api_pid}"
first_status="$?"

if [[ "${exited_pid}" == "${api_pid}" ]]; then
  api_status="${first_status}"
elif [[ "${exited_pid}" == "${worker_pid}" ]]; then
  worker_status="${first_status}"
fi

if [[ "${shutdown_requested}" -eq 0 ]]; then
  echo "A service exited unexpectedly; stopping remaining processes."
  terminate_children
fi

if [[ "${exited_pid}" != "${api_pid}" ]]; then
  wait "${api_pid}"
  api_status="$?"
fi
if [[ "${exited_pid}" != "${worker_pid}" ]]; then
  wait "${worker_pid}"
  worker_status="$?"
fi
set -e

if [[ "${first_status}" -ne 0 ]]; then
  exit "${first_status}"
fi
if [[ "${api_status}" -ne 0 ]]; then
  exit "${api_status}"
fi
if [[ "${worker_status}" -ne 0 ]]; then
  exit "${worker_status}"
fi
