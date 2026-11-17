#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

KEEP=${BACKUP_KEEP:-1}
OUT=backups
SRC=data/state.json

[ -f "$SRC" ] || { echo "no $SRC to back up" >&2; exit 1; }

mkdir -p "$OUT"
DEST="$OUT/state-$(date +%Y%m%d-%H%M%S).json"
cp "$SRC" "$DEST"

# The privacy policy caps the reign ledger at fifty entries. Keeping snapshots
# without limit would quietly defeat that, so only the newest few survive.
ls -1t "$OUT"/state-*.json 2>/dev/null | tail -n +$((KEEP + 1)) | while read -r old; do
	rm -f "$old"
done

echo "$DEST"
