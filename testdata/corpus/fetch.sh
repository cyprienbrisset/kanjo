#!/usr/bin/env bash
# Récupération reproductible du corpus de test officiel (§18.2). Ces fichiers ne sont PAS
# vendorés dans le dépôt (licences EUPL / mustangproject) : ils sont téléchargés à la demande.
#
# Sources :
#   - ConnectingEurope/eInvoicing-EN16931 (artefacts CEN, EUPL 1.2) : cas valides (examples)
#     et cas d'erreur unitaires par règle (test/*-unit-UBL/BR-*.xml).
#   - ZUGFeRD/mustangproject : Factur-X/ZUGFeRD PDF réels (divers profils).
#
# Usage : bash testdata/corpus/fetch.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
DEST="$ROOT/official"
API="https://api.github.com/repos/ConnectingEurope/eInvoicing-EN16931/contents"

dl_dir() { # <repo-path> <local-subdir>
  local path="$1" sub="$2"
  mkdir -p "$DEST/$sub"
  echo "→ $sub"
  curl -s "$API/$path" \
   | python3 -c 'import sys,json;[print(x["download_url"],x["name"]) for x in json.load(sys.stdin) if x["name"].lower().endswith(".xml")]' \
   | while read -r url name; do curl -sL -o "$DEST/$sub/$name" "$url"; done
}

dl_dir "ubl/examples"              "en16931-valid/ubl"
dl_dir "cii/examples"             "en16931-valid/cii"
dl_dir "test/Invoice-unit-UBL"    "en16931-unit/invoice-ubl"
dl_dir "test/CreditNote-unit-UBL" "en16931-unit/creditnote-ubl"

# Empreintes pour reproductibilité / intégrité.
( cd "$DEST" && find . -name '*.xml' -o -name '*.pdf' | sort | xargs shasum -a 256 > SHA256SUMS 2>/dev/null || true )
echo "Corpus téléchargé dans $DEST ($(find "$DEST" -name '*.xml' | wc -l | tr -d ' ') fichiers XML)."
