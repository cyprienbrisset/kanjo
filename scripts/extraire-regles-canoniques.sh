#!/usr/bin/env bash
# Régénère testdata/en16931/canonical-rule-ids.txt — la liste canonique des identifiants de règles
# EN 16931, extraite des Schematron OFFICIELS préprocessés (UBL + CII) du CEN. C'est la source de
# vérité du test de parité (test/parity_test.go), qui mesure notre couverture réelle du référentiel.
#
# Usage : bash scripts/extraire-regles-canoniques.sh
set -euo pipefail
cd "$(dirname "$0")/.."
base="https://raw.githubusercontent.com/ConnectingEurope/eInvoicing-EN16931/master"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "$tmp/ubl.sch" "$base/ubl/schematron/preprocessed/EN16931-UBL-validation-preprocessed.sch"
curl -fsSL -o "$tmp/cii.sch" "$base/cii/schematron/preprocessed/EN16931-CII-validation-preprocessed.sch"

ids="$(grep -rhoE 'id="BR(-[A-Z]{1,3})?-[0-9]{1,3}"' "$tmp"/*.sch | sed -E 's/id="([^"]+)"/\1/' | sort -u)"
n="$(echo "$ids" | grep -cE '^BR')"

out="testdata/en16931/canonical-rule-ids.txt"
{
  echo "# Liste canonique des identifiants de règles EN 16931 (source de vérité pour la parité)."
  echo "# Extraite des Schematron OFFICIELS préprocessés (UBL + CII) du CEN :"
  echo "#   ConnectingEurope/eInvoicing-EN16931 (EUPL 1.2)."
  echo "# Régénérer : scripts/extraire-regles-canoniques.sh"
  echo "# Total : $n règles."
  echo "$ids"
} > "$out"
echo "Écrit $out ($n règles canoniques)."
