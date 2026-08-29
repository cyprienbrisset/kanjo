# Rapport de qualité et de conformité — Kanjō

> Ce document explique **comment nous garantissons que Kanjō est digne de confiance** pour
> manipuler, convertir et valider des factures électroniques. Tous les chiffres cités sont
> **reproductibles** : lancez `scripts/rapport-qualite.sh` pour les rejouer sur votre machine.

**Dernière mise à jour** : 2026-08-29 · **Version des règles** : 2026.3

---

## 1. En bref

| Indicateur | Valeur |
|---|---|
| Fonctions de test automatisées | **174** (53 fichiers) |
| Paquets couverts par des tests | **34** |
| Règles de validation EN 16931 | **82** (78 EN 16931 · 2 CIUS FR · 2 Kanjō) |
| Corpus publiable — cas de succès | **38 / 38** déclarés conformes |
| Corpus publiable — cas d'erreur | **12 / 12** correctement rejetés |
| Corpus **réel open-source** (robustesse) | **507 documents** lus sans panic |
| Exemples réels complets (CEN + Peppol) | **40 / 40** conformes |
| Corpus **officiel Peppol** BIS 3.0 | **9 / 9** conformes |
| Cibles de compilation vérifiées | **6** (Linux/macOS/Windows × amd64/arm64), `CGO_ENABLED=0` |
| Appels réseau / télémétrie | **0** — 100 % hors-ligne |

---

## 2. Principes de conception qui préviennent les erreurs

La justesse ne s'ajoute pas après coup ; elle est **inscrite dans les types** :

- **Aucun montant en `float64`.** Les montants utilisent un type exact `Amount` (entier en unités
  mineures + échelle + devise) avec un arrondi normalisé EN 16931 (*half-away-from-zero*) calculé
  en `math/big`. Les erreurs d'arrondi flottant sont **structurellement impossibles**.
- **Aucune date en `time.Time`.** Une date de facture est un `Date{Année, Mois, Jour}` sans fuseau
  horaire : pas de décalage possible entre deux régions.
- **Les unités mineures dépendent de la devise** (0 pour JPY/HUF, 3 pour KWD…), pas d'hypothèse
  « toujours 2 décimales ».
- **« Absent » ≠ « vide ».** Les champs optionnels sont des pointeurs : on ne confond jamais une
  valeur à zéro avec une valeur manquante.
- **Les codes sont des types nommés** (`TaxCategoryCode`, `UnitCode`…) validés contre les listes
  officielles — jamais des chaînes libres.

---

## 3. Comment nous testons

### 3.1 Tests unitaires (174 fonctions)
Chaque paquet du cœur — modèle, arithmétique, lecteurs, écrivains, moteur de règles — possède ses
tests. Le calcul monétaire, les arrondis et les conversions de devises sont testés au centime près.

### 3.2 Aller-retour sans perte (*lossless*)
Pour CII et UBL, on écrit un document puis on le relit : le modèle obtenu doit être **identique**
à l'original (comparaison JSON). Cela garantit qu'aucune donnée n'est perdue silencieusement,
y compris les cas fins (remises/charges de **ligne** BG-27/28, identifiants de partie, multi-TVA).

### 3.3 Corpus publiable synthétique — *test vivant*
`testdata/corpus/published/` contient **50 factures 100 % synthétiques** (aucune donnée réelle,
donc librement redistribuables), produites de façon **déterministe** :

- **38 cas de succès** : 6 scénarios (simple, multi-TVA, avoir, autoliquidation,
  intracommunautaire, acompte) × 2 formats (CII, UBL) × 3 exemplaires, plus 2 gabarits **FatturaPA** ;
- **12 cas d'erreur** : factures volontairement non conformes.

Le test d'intégration `test/corpus_test.go` **échoue la CI** si une facture valide devient
non conforme ou si une facture erronée passe à travers les mailles. Régénérable via
`testdata/corpus/generer-corpus.sh`.

### 3.4 Corpus réel open-source (> 500 documents)
`testdata/corpus/fetch.sh` moissonne, à la demande et de façon reproductible, **plus de 500
documents réels** issus de dépôts publics (non vendorés pour raisons de licence) :

- **CEN EN 16931** (ConnectingEurope, EUPL) : exemples UBL/CII + cas unitaires par règle ;
- **XRechnung** (itplr-kosit, Apache) : jeux d'essai (testsuite + configuration du validateur) ;
- **Peppol BIS 3.0** (OpenPeppol, Apache) : exemples officiels.

Deux garanties en découlent, mesurées par `test/realcorpus_test.go` :

- **Robustesse** : les **507 documents** réels sont lus **sans jamais faire paniquer** un lecteur
  (les entrées inattendues deviennent des erreurs de fichier, jamais un crash).
- **Conformité des exemples complets** : **40/40** factures d'exemple (CEN UBL 16/16, CEN CII
  15/15, Peppol 9/9) sont validées conformes. Les cas unitaires du CEN sont des fragments par
  règle, volontairement partiels, et servent la robustesse plutôt que la validation document.

### 3.5 Corpus d'attaque (sécurité)
`testdata/fuzz/xxe/` contient des charges malveillantes (entités externes XXE, DTD distante,
*billion laughs*). Le paquet `internal/xmlsafe` les neutralise ; tout XML passe obligatoirement
par cette couche — aucun `encoding/xml` direct n'est autorisé dans les lecteurs.

---

## 4. Conformité EN 16931 — « jamais de verdict non calculé »

Kanjō applique un **moteur de règles natif en Go** (pas de Schematron, pas de JVM). Le catalogue
complet des **82 règles** est publié et généré depuis le code dans [`rules.md`](rules.md) ; il
couvre les familles `BR`, `BR-CO`, `BR-CL`, `BR-DEC`, `BR-S/Z/E/AE/K/G/O`, la CIUS française et
des règles maison (IBAN mod-97, cohérence des dates).

Chaque règle est accompagnée d'**au moins un test passant et un test échouant**. Un test de CI
vérifie que `rules.md` reste **synchronisé** avec le code : la documentation ne peut pas mentir.

Règle d'or (§17.7 du cahier des charges) : **Kanjō ne produit jamais un verdict de conformité
qui n'a pas été effectivement calculé.** Aucun sceau « conforme » n'est apposé sans exécution du
jeu de règles ; aucune conformité PDF/A n'est déclarée sans validation effective.

---

## 5. Sécurité et confidentialité

- **Aucun appel réseau** par défaut, **aucune télémétrie, jamais.** Les données de facturation ne
  quittent pas la machine.
- **Durcissement XML** systématique (anti-XXE, refus de DOCTYPE, bornes anti-bombe).
- **Écriture atomique** (fichier temporaire + renommage) ; jamais d'écriture dans le répertoire
  source.
- **Bibliothèque locale** (index SQLite) : ne stocke que des **métadonnées et empreintes**, pas le
  contenu des factures ; **purgeable** (droit à l'effacement RGPD) et à rétention configurable.
- **Journal d'audit sans donnée personnelle.**
- Binaire cœur **pur Go, `CGO_ENABLED=0`** : surface d'attaque réduite, reproductible sur 6 cibles.

Pour toute question de sécurité, conformité ou données sensibles, se référer au responsable de la
sécurité de votre organisation.

---

## 6. Reproduire ce rapport

```bash
# Tout rejouer (compilation 6 cibles, tests, règles, corpus, sécurité) :
scripts/rapport-qualite.sh

# Régénérer le corpus publiable :
testdata/corpus/generer-corpus.sh

# (Optionnel) télécharger le corpus officiel CEN :
testdata/corpus/fetch.sh

# Lancer uniquement les tests :
go test ./...
```

---

## 7. Portée et limites (en toute transparence)

Nous préférons annoncer ce qui **n'est pas** encore garanti plutôt que de le laisser deviner :

- La **validation PDF/A-3b** est **effective** via veraPDF (`kanjo embed --verify-pdfa`) : quand
  l'outil est présent, le verdict est réel ; en son absence, Kanjō ne déclare **pas** la conformité
  (il ne la simule jamais). La génération d'un PDF/A-3b pleinement conforme reste en cours de
  durcissement (OutputIntent, XMP fx:).
- Le format **EDIFACT** en lecture n'est pas encore couvert (feuille de route). **FatturaPA**
  (v1.2) est lu ; son **écriture** n'est pas encore fournie.
- La couverture de tests moyenne (~70 %) progresse lot par lot ; les chemins critiques (modèle,
  règles, conversion, lecture Factur-X) sont les mieux couverts.

Ces limites sont suivies dans le [CHANGELOG](../CHANGELOG.md) et le suivi du projet.
