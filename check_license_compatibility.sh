#!/usr/bin/env bash
# check_license_compatibility.sh
# Usage: ./check_license_compatibility.sh <license_report> <compatible_licenses_regex> <type>
# <license_report>: Path to the license report (CSV for Go, JSON for npm)
# <compatible_licenses_regex>: Regex of compatible licenses (e.g., 'GPL-3.0|MIT|BSD')
# <type>: 'go' or 'npm'

set -e

REPORT_FILE="$1"
COMPATIBLE_LICENSES="$2"
TYPE="$3"

if [[ -z "$REPORT_FILE" || -z "$COMPATIBLE_LICENSES" || -z "$TYPE" ]]; then
  echo "Usage: $0 <license_report> <compatible_licenses_regex> <type>"
  exit 2
fi

if [[ "$TYPE" == "go" ]]; then
  echo "Checking Go dependency licenses for compatibility..."
  INCOMPATIBLE=$(awk -F, 'NR>1 {print $3}' "$REPORT_FILE" | grep -Ev "$COMPATIBLE_LICENSES" | sort | uniq)
  if [ -n "$INCOMPATIBLE" ]; then
    echo "::error::Found incompatible Go dependency licenses:"
    echo "$INCOMPATIBLE"
    exit 1
  else
    echo "All Go dependency licenses are compatible."
  fi
elif [[ "$TYPE" == "npm" ]]; then
  echo "Checking npm dependency licenses for compatibility (CSV format)..."
  # npm CSV: module name,license,repository
  INCOMPATIBLE=$(awk -F, 'NR>1 {print $2}' "$REPORT_FILE" | grep -Ev "$COMPATIBLE_LICENSES" | sort | uniq)
  if [ -n "$INCOMPATIBLE" ]; then
    echo "::error::Found incompatible npm dependency licenses:"
    echo "$INCOMPATIBLE"
    exit 1
  else
    echo "All npm dependency licenses are compatible."
  fi
else
  echo "Unknown type: $TYPE. Use 'go' or 'npm'."
  exit 2
fi
