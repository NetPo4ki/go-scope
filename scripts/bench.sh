#!/usr/bin/env bash
# Reproducible benchmark run for thesis evaluation.
# Usage: ./scripts/bench.sh
# Optional: COUNT=20 OUT=benchmarks/results ./scripts/bench.sh

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

COUNT="${COUNT:-20}"
OUT="${OUT:-$ROOT/benchmarks/results}"
RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT"

if [[ -z "${GOMAXPROCS:-}" ]]; then
  if command -v getconf >/dev/null 2>&1; then
    GOMAXPROCS="$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 8)"
  else
    GOMAXPROCS="8"
  fi
  export GOMAXPROCS
fi

OUTFILE="$OUT/bench-${RUN_ID}.txt"

echo "# go-scope benchmarks" >"$OUTFILE"
echo "# date(UTC): $RUN_ID  GOMAXPROCS=$GOMAXPROCS  COUNT=$COUNT" >>"$OUTFILE"
go version >>"$OUTFILE"
echo >>"$OUTFILE"

go test \
  -bench=. \
  -benchmem \
  -count="$COUNT" \
  -timeout=60m \
  ./benchmarks/micro/... \
  ./benchmarks/macro/... \
  2>&1 | tee -a "$OUTFILE"

echo >>"$OUTFILE"
echo "Wrote $OUTFILE"
