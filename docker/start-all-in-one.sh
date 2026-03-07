#!/usr/bin/env bash
set -euo pipefail

export SERVER_PORT="${SERVER_PORT:-${API_PORT:-8080}}"
export SERVER_WEB_DIST_DIR="${SERVER_WEB_DIST_DIR:-${WEB_DIST_DIR:-/app/web-dist}}"
export REDIS_ADDR="${REDIS_ADDR:-redis:6379}"
export ASYNQ_CONCURRENCY="${ASYNQ_CONCURRENCY:-${WORKER_CONCURRENCY:-10}}"
export PYTHON_BIN="${PYTHON_BIN:-python3}"
export PYTHON_SCRIPT="${PYTHON_SCRIPT:-/app/scripts/python/account_manager.py}"
export PYTHON_TIMEOUT_SECONDS="${PYTHON_TIMEOUT_SECONDS:-${PYTHON_TIMEOUT_SEC:-60}}"
export PATHS_OCTO_MODULE_DIR="${PATHS_OCTO_MODULE_DIR:-${OCTO_MODULE_DIR:-/app/scripts/python/modules}}"
export LOGGING_FILE="${LOGGING_FILE:-${LOG_FILE:-/app/logs/octomanger.log}}"
export LOGGING_LEVEL="${LOGGING_LEVEL:-${LOG_LEVEL:-info}}"
export DATABASE_DSN="${DATABASE_DSN:-${DATABASE_URL:-}}"
export DATABASE_URL="${DATABASE_URL:-}"
export DATABASE_AUTO_MIGRATE="${DATABASE_AUTO_MIGRATE:-true}"
export DATABASE_RESET="${DATABASE_RESET:-false}"

REDIS_HOST="${REDIS_ADDR%:*}"
REDIS_PORT="${REDIS_ADDR##*:}"

db_conn="${DATABASE_DSN}"
if [[ -z "${db_conn}" ]]; then
  db_conn="${DATABASE_URL}"
fi

wait_for_tcp() {
  local host="$1"
  local port="$2"
  local service_name="$3"
  local max_attempts="${4:-60}"
  local sleep_sec="${5:-2}"

  echo "Waiting for ${service_name}..."
  for i in $(seq 1 "${max_attempts}"); do
    if (echo >"/dev/tcp/${host}/${port}") >/dev/null 2>&1; then
      echo "${service_name} is ready."
      return 0
    fi
    if [[ "${i}" -eq "${max_attempts}" ]]; then
      echo "${service_name} is not reachable."
      return 1
    fi
    sleep "${sleep_sec}"
  done
}

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

wait_for_tcp "${REDIS_HOST}" "${REDIS_PORT}" "Redis"

if [[ -n "${db_conn}" ]]; then
  wait_for_postgres "${db_conn}"
else
  echo "DATABASE_DSN/DATABASE_URL is empty; PostgreSQL checks are skipped."
fi

/app/api &
api_pid=$!
DATABASE_AUTO_MIGRATE=false /app/worker &
worker_pid=$!

trap 'kill -TERM "${api_pid}" "${worker_pid}" 2>/dev/null || true; wait || true' SIGINT SIGTERM

wait -n "${api_pid}" "${worker_pid}"
status=$?
kill -TERM "${api_pid}" "${worker_pid}" 2>/dev/null || true
wait || true
exit "${status}"
