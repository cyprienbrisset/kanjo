#!/usr/bin/env bash
# Génère le corpus de test PUBLIABLE de Kanjō — 100 % synthétique, déterministe (graines fixes),
# sans aucune donnée réelle ni tierce (donc librement redistribuable).
#
# À la différence de `fetch.sh` (qui télécharge le corpus officiel CEN, non vendoré pour raisons
# de licence), ce corpus-ci est produit par `kanjo generate` et versionné dans le dépôt.
#
# Usage : testdata/corpus/generer-corpus.sh [chemin-du-binaire-kanjo]
set -euo pipefail

KANJO="${1:-kanjo}"
ROOT="$(cd "$(dirname "$0")" && pwd)/published"

rm -rf "$ROOT"
mkdir -p "$ROOT"

SCENARIOS=(simple multi-tva avoir autoliquidation intracommunautaire acompte)
FORMATS=(cii ubl)

seed=100
# --- Cas de SUCCÈS : chaque scénario, chaque format ---
for sc in "${SCENARIOS[@]}"; do
  for fmt in "${FORMATS[@]}"; do
    "$KANJO" generate --count 2 --scenario "$sc" --format "$fmt" --seed "$seed" \
      --out "$ROOT/valides/$fmt/$sc" >/dev/null
    seed=$((seed + 1))
  done
done

# --- Cas d'ERREUR : factures volontairement non conformes ---
for fmt in "${FORMATS[@]}"; do
  "$KANJO" generate --count 5 --invalid --format "$fmt" --seed "$seed" \
    --out "$ROOT/invalides/$fmt" >/dev/null
  seed=$((seed + 1))
done

echo "Corpus publiable généré dans $ROOT"
find "$ROOT" -type f | wc -l | xargs echo "Fichiers :"
