#!/bin/sh
# Load the 14-segment alpha font (U+E000–U+E019) then run the viewer.
root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
export FONTCONFIG_FILE="$root/fonts.conf"
cd "$root" || exit 1
exec go run .
