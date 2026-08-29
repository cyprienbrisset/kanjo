#!/usr/bin/env bash
# Récupération reproductible d'un large corpus de test RÉEL et OPEN-SOURCE (§18.2). Ces fichiers
# ne sont PAS vendorés (licences EUPL / Apache / CC) : ils sont moissonnés à la demande depuis des
# dépôts publics via l'API GitHub « git/trees » (récursive, 1 requête par dépôt) puis téléchargés
# depuis raw.githubusercontent.com (hors quota API). Le corpus synthétique redistribuable, lui,
# est versionné sous published/.
#
# Sources (toutes publiques, ≈ 7 400 documents RÉELS) :
#   - ConnectingEurope/eInvoicing-EN16931       (CEN, EUPL 1.2) : exemples + cas unitaires UBL/CII
#   - itplr-kosit/xrechnung-testsuite            (Apache-2.0)    : jeux d'essai XRechnung
#   - itplr-kosit/validator-configuration-xrechnung (Apache-2.0) : cas du validateur XRechnung
#   - OpenPeppol/peppol-bis-invoice-3            (Apache-2.0)    : corpus Peppol BIS 3.0
#   - phax/phive-rules                           (Apache-2.0)    : ~6 400 instances de test
#                                                                  multi-juridictions (Peppol, EN 16931,
#                                                                  XRechnung, FatturaPA, A-NZ, SG…)
#   - ZUGFeRD/corpus, ZUGFeRD/mustangproject     (divers OSS)    : Factur-X / ZUGFeRD réels
#
# Usage : bash testdata/corpus/fetch.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
DEST="$ROOT/official"

# Jeton GitHub (facultatif mais recommandé) : porte la limite API de 60/h à 5000/h.
# Utilise GITHUB_TOKEN, sinon le jeton de la CLI gh si disponible.
TOKEN="${GITHUB_TOKEN:-$(gh auth token 2>/dev/null || true)}"
gh_api() { # <url>
  if [ -n "$TOKEN" ]; then curl -fsS -H "Authorization: Bearer $TOKEN" "$1"
  else curl -fsS "$1"; fi
}

# harvest <repo> <prefix-de-chemin> <sous-dossier-local>
# Télécharge tous les .xml du dépôt dont le chemin commence par <prefix>.
harvest() {
  local repo="$1" prefix="$2" sub="$3"
  local branch
  branch="$(gh_api "https://api.github.com/repos/$repo" \
    | python3 -c 'import sys,json;print(json.load(sys.stdin).get("default_branch","master"))' 2>/dev/null || echo master)"
  local dir="$DEST/$sub"
  mkdir -p "$dir"
  echo "→ $sub  ($repo@$branch : $prefix)"
  # Émet des paires « url destination » (sans espace) puis télécharge en parallèle (xargs -P 16).
  gh_api "https://api.github.com/repos/$repo/git/trees/$branch?recursive=1" \
    | REPO="$repo" BRANCH="$branch" PREFIX="$prefix" DIR="$dir" python3 -c '
import sys, json, os, urllib.parse
prefix = os.environ.get("PREFIX", "")
repo, branch, d = os.environ["REPO"], os.environ["BRANCH"], os.environ["DIR"]
try:
    tree = json.load(sys.stdin).get("tree", [])
except Exception:
    tree = []
for e in tree:
    p = e.get("path", "")
    if p.lower().endswith(".xml") and p.startswith(prefix):
        url = "https://raw.githubusercontent.com/%s/%s/%s" % (repo, branch, urllib.parse.quote(p))
        dest = os.path.join(d, p.replace("/", "_").replace(" ", "_"))
        sys.stdout.write(url + "\0" + dest + "\0")' \
    | xargs -0 -P 10 -n2 sh -c 'curl -fsSL -o "$2" "$1" || true' _
  echo "   $(find "$dir" -name '*.xml' | wc -l | tr -d ' ') fichiers"
}

harvest "ConnectingEurope/eInvoicing-EN16931"       "ubl/examples"   "cen/ubl-examples"
harvest "ConnectingEurope/eInvoicing-EN16931"       "cii/examples"   "cen/cii-examples"
harvest "ConnectingEurope/eInvoicing-EN16931"       "test/"          "cen/unit"
harvest "itplr-kosit/xrechnung-testsuite"           ""               "xrechnung/testsuite"
harvest "itplr-kosit/validator-configuration-xrechnung" ""           "xrechnung/validator"
harvest "OpenPeppol/peppol-bis-invoice-3"           ""               "peppol-bis"
harvest "phax/phive-rules"                          ""               "phive-rules"
harvest "ZUGFeRD/corpus"                            ""               "zugferd/corpus"
harvest "ZUGFeRD/mustangproject"                    ""               "zugferd/mustang"

# Empreintes pour reproductibilité / intégrité.
( cd "$DEST" && find . -name '*.xml' -o -name '*.pdf' | sort | xargs shasum -a 256 > SHA256SUMS 2>/dev/null || true )
echo "Corpus téléchargé dans $DEST ($(find "$DEST" -name '*.xml' | wc -l | tr -d ' ') fichiers XML)."
