#!/usr/bin/env bash
# Récupération reproductible d'un large corpus de test RÉEL et OPEN-SOURCE (§18.2). Ces fichiers
# ne sont PAS vendorés (licences EUPL / Apache / CC) : ils sont moissonnés à la demande depuis des
# dépôts publics via l'API GitHub « git/trees » (récursive, 1 requête par dépôt) puis téléchargés
# depuis raw.githubusercontent.com (hors quota API). Le corpus synthétique redistribuable, lui,
# est versionné sous published/.
#
# Sources (toutes publiques) :
#   - ConnectingEurope/eInvoicing-EN16931   (CEN, EUPL 1.2)  : exemples + cas unitaires UBL/CII
#   - itplr-kosit/xrechnung-testsuite        (Apache-2.0)     : jeux d'essai XRechnung
#   - OpenPeppol/peppol-bis-invoice-3        (Apache-2.0)     : exemples Peppol BIS 3.0
#   - phax/en16931-ubl-example / cii-example (Apache-2.0)     : exemples UBL/CII
#
# Usage : bash testdata/corpus/fetch.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
DEST="$ROOT/official"

# harvest <repo> <prefix-de-chemin> <sous-dossier-local>
# Télécharge tous les .xml du dépôt dont le chemin commence par <prefix>.
harvest() {
  local repo="$1" prefix="$2" sub="$3"
  local branch
  branch="$(curl -fsS "https://api.github.com/repos/$repo" \
    | python3 -c 'import sys,json;print(json.load(sys.stdin).get("default_branch","master"))' 2>/dev/null || echo master)"
  mkdir -p "$DEST/$sub"
  echo "→ $sub  ($repo@$branch : $prefix)"
  local n=0
  while read -r p; do
    [ -z "$p" ] && continue
    local fn; fn="$(echo "$p" | tr '/' '_')"
    if curl -fsSL -o "$DEST/$sub/$fn" "https://raw.githubusercontent.com/$repo/$branch/$p"; then
      n=$((n + 1))
    fi
  done < <(curl -fsS "https://api.github.com/repos/$repo/git/trees/$branch?recursive=1" \
    | PREFIX="$prefix" python3 -c '
import sys, json, os
prefix = os.environ.get("PREFIX", "")
try:
    tree = json.load(sys.stdin).get("tree", [])
except Exception:
    tree = []
for e in tree:
    p = e.get("path", "")
    if p.lower().endswith(".xml") and p.startswith(prefix):
        print(p)')
  echo "   $n fichiers"
}

harvest "ConnectingEurope/eInvoicing-EN16931"       "ubl/examples"   "cen/ubl-examples"
harvest "ConnectingEurope/eInvoicing-EN16931"       "cii/examples"   "cen/cii-examples"
harvest "ConnectingEurope/eInvoicing-EN16931"       "test/"          "cen/unit"
harvest "itplr-kosit/xrechnung-testsuite"           ""               "xrechnung/testsuite"
harvest "itplr-kosit/validator-configuration-xrechnung" ""           "xrechnung/validator"
harvest "OpenPeppol/peppol-bis-invoice-3"           "rules/examples" "peppol-bis"

# Empreintes pour reproductibilité / intégrité.
( cd "$DEST" && find . -name '*.xml' -o -name '*.pdf' | sort | xargs shasum -a 256 > SHA256SUMS 2>/dev/null || true )
echo "Corpus téléchargé dans $DEST ($(find "$DEST" -name '*.xml' | wc -l | tr -d ' ') fichiers XML)."
