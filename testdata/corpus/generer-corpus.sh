#!/usr/bin/env bash
# Génère le corpus de test PUBLIABLE de Kanjō — 100 % synthétique, déterministe (graines fixes),
# sans aucune donnée réelle ni tierce (donc librement redistribuable).
#
# À la différence de `fetch.sh` (qui télécharge le corpus officiel CEN, non vendoré pour raisons
# de licence), ce corpus-ci est produit par `kanjo generate` (+ quelques gabarits FatturaPA) et
# versionné dans le dépôt.
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
# --- Cas de SUCCÈS générés : chaque scénario, chaque format ---
for sc in "${SCENARIOS[@]}"; do
  for fmt in "${FORMATS[@]}"; do
    "$KANJO" generate --count 3 --scenario "$sc" --format "$fmt" --seed "$seed" \
      --out "$ROOT/valides/$fmt/$sc" >/dev/null
    seed=$((seed + 1))
  done
done

# --- Cas d'ERREUR générés : factures volontairement non conformes ---
for fmt in "${FORMATS[@]}"; do
  "$KANJO" generate --count 6 --invalid --format "$fmt" --seed "$seed" \
    --out "$ROOT/invalides/$fmt" >/dev/null
  seed=$((seed + 1))
done

# --- Gabarits FatturaPA (le générateur ne produit pas ce format) ---
mkdir -p "$ROOT/valides/fatturapa"
for n in 0001 0002; do
  cat > "$ROOT/valides/fatturapa/F2026-$n.xml" <<XML
<?xml version="1.0" encoding="UTF-8"?>
<p:FatturaElettronica versione="FPR12" xmlns:p="http://ivaservizi.agenziaentrate.gov.it/docs/xsd/fatture/v1.2">
 <FatturaElettronicaHeader>
  <CedentePrestatore><DatiAnagrafici><IdFiscaleIVA><IdPaese>IT</IdPaese><IdCodice>1234567890$n</IdCodice></IdFiscaleIVA><Anagrafica><Denominazione>Rossi SRL</Denominazione></Anagrafica></DatiAnagrafici><Sede><Indirizzo>Via Roma 1</Indirizzo><CAP>00100</CAP><Comune>Roma</Comune><Nazione>IT</Nazione></Sede></CedentePrestatore>
  <CessionarioCommittente><DatiAnagrafici><IdFiscaleIVA><IdPaese>IT</IdPaese><IdCodice>9876543210$n</IdCodice></IdFiscaleIVA><Anagrafica><Denominazione>Bianchi SPA</Denominazione></Anagrafica></DatiAnagrafici><Sede><Indirizzo>Corso Milano 2</Indirizzo><CAP>20100</CAP><Comune>Milano</Comune><Nazione>IT</Nazione></Sede></CessionarioCommittente>
 </FatturaElettronicaHeader>
 <FatturaElettronicaBody>
  <DatiGenerali><DatiGeneraliDocumento><TipoDocumento>TD01</TipoDocumento><Divisa>EUR</Divisa><Data>2026-08-12</Data><Numero>IT2026-$n</Numero><ImportoTotaleDocumento>122.00</ImportoTotaleDocumento></DatiGeneraliDocumento></DatiGenerali>
  <DatiBeniServizi><DettaglioLinee><NumeroLinea>1</NumeroLinea><Descrizione>Consulenza</Descrizione><Quantita>1.00</Quantita><PrezzoUnitario>100.00</PrezzoUnitario><PrezzoTotale>100.00</PrezzoTotale><AliquotaIVA>22.00</AliquotaIVA></DettaglioLinee><DatiRiepilogo><AliquotaIVA>22.00</AliquotaIVA><ImponibileImporto>100.00</ImponibileImporto><Imposta>22.00</Imposta></DatiRiepilogo></DatiBeniServizi>
  <DatiPagamento><DettaglioPagamento><ModalitaPagamento>MP05</ModalitaPagamento><DataScadenzaPagamento>2026-09-11</DataScadenzaPagamento><ImportoPagamento>122.00</ImportoPagamento><IBAN>IT60X0542811101000000123456</IBAN></DettaglioPagamento></DatiPagamento>
 </FatturaElettronicaBody>
</p:FatturaElettronica>
XML
done

echo "Corpus publiable généré dans $ROOT"
find "$ROOT" -type f | wc -l | xargs echo "Fichiers :"
