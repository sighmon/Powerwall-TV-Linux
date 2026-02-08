#!/usr/bin/env bash
set -euo pipefail
FILE="vendor/github.com/gotk3/gotk3/gdk/gdk_since_3_22.go"
if ! grep -q "internal/callback" "$FILE"; then
  # insert callback import after glib import
  awk 'BEGIN{added=0} {print} /github.com\/gotk3\/gotk3\/glib/ && !added {print "\t\"github.com/gotk3/gotk3/internal/callback\""; added=1}' "$FILE" > /tmp/gdk_since_3_22.go && mv /tmp/gdk_since_3_22.go "$FILE"
fi
