#!/usr/bin/env bash
# Rejoue l'ensemble des vérifications qualité de Kanjō et imprime un rapport synthétique.
# Objectif : permettre à quiconque de reproduire les chiffres du document docs/RAPPORT-QUALITE.md.
#
# Usage : scripts/rapport-qualite.sh
set -euo pipefail
cd "$(dirname "$0")/.."

echo "=================================================================="
echo " Rapport de qualité — Kanjō"
echo " Commit : $(git rev-parse --short HEAD 2>/dev/null || echo n/a)   Date : $(date +%F)"
echo "=================================================================="

echo
echo "## 1. Compilation multiplateforme (CGO_ENABLED=0)"
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  GOOS="${target%/*}" GOARCH="${target#*/}" CGO_ENABLED=0 go build -o /dev/null ./cmd/kanjo \
    && echo "  OK  $target"
done

echo
echo "## 2. Tests unitaires et d'intégration"
CGO_ENABLED=0 go test ./... 2>&1 | grep -E '^(ok|FAIL|---)' | sed 's/^/  /' || true
TESTS=$(grep -rhoE '^func (Test|Fuzz|Example)[A-Za-z0-9_]+' --include='*_test.go' . | wc -l | tr -d ' ')
echo "  → $TESTS fonctions de test"

echo
echo "## 3. Jeu de règles EN 16931"
grep -E '^## ' docs/rules.md | sed 's/^/  /'

echo
echo "## 4. Corpus publiable synthétique (testdata/corpus/published)"
kanjo_bin="$(command -v kanjo || echo ./kanjo)"
[ -x "$kanjo_bin" ] || { CGO_ENABLED=0 go build -o ./kanjo ./cmd/kanjo; kanjo_bin=./kanjo; }
count_ok() { find "$1" -name '*.xml' -print0 | while IFS= read -r -d '' f; do
  "$kanjo_bin" validate "$f" >/dev/null 2>&1 && echo ok || echo ko; done; }
V_OK=$(count_ok testdata/corpus/published/valides | grep -c ok || true)
V_TOT=$(find testdata/corpus/published/valides -name '*.xml' | wc -l | tr -d ' ')
I_KO=$(count_ok testdata/corpus/published/invalides | grep -c ko || true)
I_TOT=$(find testdata/corpus/published/invalides -name '*.xml' | wc -l | tr -d ' ')
echo "  valides   : $V_OK / $V_TOT déclarés conformes (attendu : tous)"
echo "  invalides : $I_KO / $I_TOT correctement rejetés (attendu : tous)"

echo
echo "## 5. Corpus officiel CEN (si présent — voir testdata/corpus/fetch.sh)"
if [ -d testdata/corpus/official/en16931-valid ]; then
  C_OK=$(count_ok testdata/corpus/official/en16931-valid | grep -c ok || true)
  C_TOT=$(find testdata/corpus/official/en16931-valid -name '*.xml' | wc -l | tr -d ' ')
  echo "  en16931-valid : $C_OK / $C_TOT conformes"
else
  echo "  (absent — exécuter testdata/corpus/fetch.sh pour le télécharger)"
fi

echo
echo "## 6. Durcissement XML (anti-XXE / anti-bombe)"
CGO_ENABLED=0 go test ./internal/xmlsafe/ 2>&1 | grep -E '^(ok|FAIL)' | sed 's/^/  /' || true

echo
echo "Rapport terminé."
