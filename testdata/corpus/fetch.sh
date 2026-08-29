#!/usr/bin/env bash
# Récupération reproductible du corpus de test officiel (§18.2). Ces fichiers ne sont PAS
# vendorés dans le dépôt (licences EUPL / Apache / mustangproject) : ils sont téléchargés à la
# demande. Le corpus 100 % synthétique et redistribuable, lui, est versionné sous published/.
#
# Sources :
#   - ConnectingEurope/eInvoicing-EN16931 (artefacts CEN, EUPL 1.2) : exemples valides UBL/CII
#     et cas d'erreur unitaires par règle.
#   - OpenPeppol/peppol-bis-invoice-3 (Apache-2.0) : exemples Peppol BIS Billing 3.0.
#   - ZUGFeRD/mustangproject : Factur-X/ZUGFeRD PDF réels (divers profils).
#
# Usage : bash testdata/corpus/fetch.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
DEST="$ROOT/official"

# dl_dir <repo> <repo-path> <local-subdir> : télécharge tous les .xml d'un dossier GitHub.
dl_dir() {
  local repo="$1" path="$2" sub="$3"
  mkdir -p "$DEST/$sub"
  echo "→ $sub  ($repo/$path)"
  curl -s "https://api.github.com/repos/$repo/contents/$path" \
   | python3 -c 'import sys,json
try:
    items=json.load(sys.stdin)
except Exception:
    sys.exit(0)
if isinstance(items,list):
    [print(x["download_url"],x["name"]) for x in items if str(x.get("name","")).lower().endswith(".xml")]' \
   | while read -r url name; do [ -n "$url" ] && curl -sL -o "$DEST/$sub/$name" "$url"; done
}

# --- CEN EN 16931 (exemples valides + cas d'erreur unitaires) ---
dl_dir "ConnectingEurope/eInvoicing-EN16931" "ubl/examples"              "en16931-valid/ubl"
dl_dir "ConnectingEurope/eInvoicing-EN16931" "cii/examples"             "en16931-valid/cii"
dl_dir "ConnectingEurope/eInvoicing-EN16931" "test/Invoice-unit-UBL"    "en16931-unit/invoice-ubl"
dl_dir "ConnectingEurope/eInvoicing-EN16931" "test/CreditNote-unit-UBL" "en16931-unit/creditnote-ubl"
dl_dir "ConnectingEurope/eInvoicing-EN16931" "test/Invoice-unit-CII"    "en16931-unit/invoice-cii"

# --- Peppol BIS Billing 3.0 (exemples officiels) ---
dl_dir "OpenPeppol/peppol-bis-invoice-3" "rules/examples" "peppol-bis/examples"

# Empreintes pour reproductibilité / intégrité.
( cd "$DEST" && find . -name '*.xml' -o -name '*.pdf' | sort | xargs shasum -a 256 > SHA256SUMS 2>/dev/null || true )
echo "Corpus téléchargé dans $DEST ($(find "$DEST" -name '*.xml' | wc -l | tr -d ' ') fichiers XML)."
