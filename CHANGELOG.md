# Changelog

Toutes les modifications notables de Kanjō sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/),
versionnage sémantique ([SemVer](https://semver.org/lang/fr/)).

Trois numéros de version sont suivis (voir §19.3 du cahier des charges) :
version de l'outil · version du jeu de règles · version du schéma de sortie.

Une modification de règle de validation qui change un verdict apparaît dans une section
dédiée **« Conformité »** et incrémente la version du jeu de règles.

## [Non publié]

### Modifié (cohérence UI)
- **Suppression des idéogrammes de l'interface CLI** (comme déjà fait dans la GUI) : marqueurs de
  statut clairs et universels `✓` / `⚠` / `✗` / `·`, en-têtes `▸`, bandeaux sans idéogramme. Le
  rapport de validation HTML et le message de lancement de Studio sont également nettoyés. Les
  commentaires de code documentant le système de design (大福帳) sont conservés.

### Assurance qualité (confiance)
- **Corpus élargi** : le corpus publiable passe à **50 factures** (38 succès dont 2 FatturaPA,
  12 erreurs) ; `fetch.sh` récupère désormais aussi les **exemples officiels Peppol BIS 3.0**
  (OpenPeppol) et les cas unitaires CII du CEN. Résultats mesurés : **Peppol BIS 9/9 conformes**,
  CEN 31/32. Corpus officiel total récupérable ≈ 318 fichiers.
- **Corpus de test publiable** ([`testdata/corpus/published`](testdata/corpus/published)) : factures
  100 % synthétiques et déterministes (24 cas de succès sur 6 scénarios × CII/UBL, 10 cas d'erreur),
  versionnées et librement redistribuables. Régénérables via `testdata/corpus/generer-corpus.sh`.
- **Test d'intégration vivant** (`test/corpus_test.go`) : la CI échoue si un cas valide devient non
  conforme ou si un cas erroné passe (24/24 conformes, 10/10 rejetés).
- **Rapport de qualité et de conformité** ([`docs/RAPPORT-QUALITE.md`](docs/RAPPORT-QUALITE.md)) :
  méthodologie de test, résultats, sécurité/RGPD, portée et limites — chiffres reproductibles via
  `scripts/rapport-qualite.sh`. Liens depuis le README et la page GitHub.

### Corpus réel (au lieu de factures fabriquées)
- **Corpus de test RÉEL porté à 7 365 documents** open-source (aucune facture générée) : `fetch.sh`
  moissonne en parallèle (authentifié, robuste aux noms de fichiers) CEN EN 16931, XRechnung,
  Peppol BIS 3.0, **phive-rules** (~6 400 instances multi-juridictions) et ZUGFeRD/mustangproject.
  `test/realcorpus_test.go` : **7 365 lus sans panic**, 4 728/7 365 lisibles, exemples CEN 32/32.

### Tests
- **Couverture maximisée** : nouveaux tests pour `pkg/write/cii` (0 → 67 %), `pkg/write/ubl`
  (0 → 68 %), `pkg/api` (0 → 100 %), `pkg/rules` moteur (47 → 91 %) et davantage de commandes
  `cmd/kanjo/cli` (19 → 33 %). **174 fonctions de test** sur **34 paquets**, couverture moyenne
  ~70 %.
- **Corpus réel > 500 documents** : `fetch.sh` moissonne désormais (API git/trees, récursif)
  **507 documents** open-source (CEN EN 16931, XRechnung testsuite + validateur, Peppol BIS 3.0).
  `test/realcorpus_test.go` garantit qu'ils sont tous **lus sans panic** (robustesse) et que les
  **40 exemples complets** sont validés conformes.
- **`pkg/read`** : test table-driven de `Detect` (CII, ZUGFeRD 1.0, UBL Invoice/CreditNote, PDF,
  JSON pivot, inconnu, gestion BOM + blancs) et rejet d'un format non reconnu par `ReadBytes`
  → couverture 0 % → **61 %**.
- **`pkg/read/facturx`** : fabrication d'un PDF Factur-X (PDF/A-3 + CII embarqué) via `pdfa.EmbedXML`
  puis relecture ; vérifie l'identifiant extrait, le marquage du format porteur et l'avertissement
  sur un nom de pièce jointe non conforme → couverture 0 % → **87 %**.
- **`pkg/convert`** : suite dédiée (aller-retour CII→UBL/JSON, cible inconnue, calcul des pertes
  W-EXT-002 et politiques `none`/`minor`/`any`/`allow-loss`) → couverture 0 % → **95 %**.
- **`cmd/kanjo/cli`** : tests de bout en bout (capture stdout/stderr + codes de sortie) pour
  `version`, `help`, commande inconnue, `convert`, `validate`, `inspect`, entrée illisible.

### Ajouté (complétude)
- **Remises et charges de ligne (BG-27/28)** : lecture et écriture complètes pour CII et UBL
  (montant BT-136/141, base, pourcentage, motif BT-139/144 et code motif BT-140/145). Tests
  d'aller-retour lossless dédiés (`TestCIILineAllowanceRoundTrip`, `TestUBLLineAllowanceRoundTrip`).

### Ajouté (nouveau format)
- **Lecture FatturaPA** (FatturaElettronica v1.2, format italien) : `pkg/read/fatturapa` — en-tête
  (cédant/cessionnaire, TVA, adresses), lignes de détail, ventilation de TVA et totaux, mappés
  vers le pivot EN 16931. Détection par le contenu (`FatturaElettronica`). Une FatturaPA peut
  ainsi être validée, convertie (UBL/CII/Factur-X) et rendue lisible comme tout autre format.

### Ajouté (conformité PDF/A)
- **Association Factur-X structurelle à l'embarquement** : `embed` établit désormais l'association
  exigée par PDF/A-3 — l'`EmbeddedFile` porte `/Subtype text/xml` et `/Params`, la spécification de
  fichier porte **`/AFRelationship /Data`**, et le catalogue référence le fichier via un tableau
  **`/AF`** (ISO 19005-3 §6.8). Vérifié par **relecture** du PDF produit (`pkg/pdfa` :
  `TestEmbedEstablishesFacturXAssociation`). L'association rend le XML *associé* au document, pas
  une simple pièce jointe.
- **Job CI veraPDF + corpus PDF/A réel** : une facture **Factur-X EN 16931 réelle** (PDF/A-3b,
  corpus akretion, BSD) est archivée dans `testdata/corpus/pdfa/` ; un job CI installe **veraPDF** et
  valide effectivement ce PDF ainsi que la sortie d'`embed`. La préservation globale de la conformité
  PDF/A-3b après réécriture est **mesurée**, jamais supposée (§17.7).
- **Validation PDF/A-3b effective** via veraPDF : `pkg/pdfa.ValidatePDFA` + option
  `kanjo embed --verify-pdfa`. Verdict réel quand veraPDF est installé ; **jamais de conformité
  déclarée sans validation** (§17.7) — en l'absence de l'outil, le rapport l'indique explicitement.

### Conformité — listes de codes ISO (parité 60 %)
- **Validation ISO effective** : devise ISO 4217 (**BR-CL-03**, **BR-CL-04**) et codes pays
  ISO 3166-1 (**BR-CL-14**), avec listes embarquées (`pkg/model/codelists.go`).
- **Correction d'ID mal étiquetés** : notre `BR-CL-03` validait le code type (c'est **BR-CL-01**)
  et notre `BR-CL-01` la devise (c'est **BR-CL-04**) — réalignés sur le Schematron officiel.

### Conformité — remises/charges par catégorie (parité 52 %)
- **24 règles TVA de niveau document** (remises BG-20 / charges BG-21) : identifiants requis
  `BR-S/Z/E/G-03/04` (vendeur), `BR-AE/IC-03/04` (vendeur + acheteur), et contraintes de taux
  `BR-S-06/07` (>0), `BR-Z/E/AE/G/IC-06/07` (=0). Parité EN 16931 **93 → 117/223 (52 %)**.

### Conformité — mesure de parité EN 16931
- **Liste canonique vendorée** (`testdata/en16931/canonical-rule-ids.txt`, **223 règles**) extraite
  du **Schematron officiel CEN** (préprocessé UBL+CII), régénérable par
  `scripts/extraire-regles-canoniques.sh`.
- **Test de parité** (`test/parity_test.go`) : mesure la couverture réelle — **91/223 (41 %)** —
  avec **cliquet anti-régression**, et génère [`docs/CONFORMITE-EN16931.md`](docs/CONFORMITE-EN16931.md)
  (couvertes/manquantes par famille). La conformité devient un chiffre **mesuré**, pas déclaré.
- **Réconciliation des identifiants** avec le référentiel officiel : famille **BR-K-\* → BR-IC-\***
  (intracommunautaire) et **renumérotation BR-DEC** alignée sur les termes BT du Schematron
  (BT-109→BR-DEC-12, BT-112→BR-DEC-14, BT-115→BR-DEC-18, BT-116/117→BR-DEC-19/20, BT-131→BR-DEC-23,
  BT-136→BR-DEC-24, BT-141→BR-DEC-27). `BR-CO-25` (hors Schematron CEN) est signalé « au-delà ».

### Conformité
- **Règles par catégorie de TVA (13 règles, jeu → 95)** : existence d'une ventilation par
  catégorie employée (**BR-Z/E/AE/K/G/O-01**), identifiant TVA/fiscal du vendeur pour les
  catégories imposables (**BR-S/Z/E/G-02**), identifiants vendeur **et** acheteur pour
  l'autoliquidation et l'intracommunautaire (**BR-AE-02**, **BR-K-02**), et absence de motif
  d'exonération au taux normal (**BR-S-10**).
- **Correctif de conformité du générateur** : les scénarios *autoliquidation* et
  *intracommunautaire* produisaient des factures **sans identifiant TVA de l'acheteur** (pourtant
  requis) ; désormais l'acheteur porte une TVA (et un pays d'un autre État membre pour
  l'intracommunautaire). Défaut mis au jour par les nouvelles règles BR-AE-02/BR-K-02.
- **Décimales des remises/charges (4 règles, jeu → 82)** : **BR-DEC-01/05** (montant de remise
  BT-92 / charge BT-99 au niveau document ≤ 2 décimales) et **BR-DEC-27/28** (remise BT-136 /
  charge BT-141 au niveau ligne). Tests passant/échouant.
- **Périodes et facture antérieure (5 règles, jeu → 78)** : **BR-CO-19/20** (une période de
  facturation ou de ligne doit porter une date de début ou de fin), **BR-29/30** (la date de fin
  ne doit pas précéder le début) et **BR-55** (chaque référence de facture antérieure BG-3 doit
  porter un identifiant BT-25). Tests passant/échouant ; corpus inchangé.
- **16 règles EN 16931** précédentes (jeu de règles → 73) :
  - Totaux du document (BG-22) : **BR-12** (total des lignes BT-106), **BR-13** (total HT
    BT-109), **BR-14** (total TTC BT-112) et **BR-15** (net à payer BT-115) obligatoires.
  - Remises/charges — niveau document : **BR-31/32/33** (remise : montant, catégorie de TVA,
    motif) et **BR-36/37/38** (charge : montant, catégorie de TVA, motif).
  - Remises/charges — niveau ligne : **BR-41/42** (remise : montant, motif) et **BR-43/44**
    (charge : montant, motif).
  - Ventilation de TVA (BG-23) : **BR-45** (montant imposable BT-116) et **BR-46** (montant de
    TVA BT-117) obligatoires par ventilation.
  - Chaque règle est couverte par un test passant et un test échouant. Conformité du corpus
    officiel CEN inchangée (seul le cas hors-norme catégorie « B » reste correctement rejeté).

### Corrigé
- **Client lourd — bouton « Choisir un fichier »** : WKWebView ne fournit pas de sélecteur
  de fichiers pour `<input type=file>`. Ajout d'un pont natif `window.kanjoOpenFiles`
  (`osascript` sur macOS, `zenity` sur Linux, `OpenFileDialog` PowerShell sur Windows) qui
  ouvre le dialogue système et transmet nom + contenu (base64) au frontend. Le mode navigateur
  (`kanjo studio`) reste inchangé. Aucune dépendance supplémentaire.

### Ajouté
- Bootstrap du dépôt : `go.mod`, arborescence du §6, `README.md`.
  ADR-011 (Q7) : Kanjō est un outil autonome.
- **Modèle pivot** (`pkg/model`) : types exacts `Amount` (unités mineures, arrondi EN 16931
  half-away-from-zero via `math/big`), `Decimal`, `Date` (sans fuseau), codes nommés
  (`TypeCode`, `TaxCategoryCode`, `PaymentMeansCode`, `UnitCode`) avec `Valid()`/`Label()`,
  `Document`/`Party`/`Line`/`TaxSubtotal`/`Totals`, `Extensions` typées, `Provenance`.
  Tests de propriétés (associativité, absence de dérive sur 10 000 opérations).
- **XML durci** (`internal/xmlsafe`) : seul point d'entrée XML ; rejet DOCTYPE/DTD, entités
  externes, bornes de profondeur/nœuds/taille, jeux de caractères sûrs (UTF-8/Latin-1/CP1252).
  Batterie d'attaques XXE / billion laughs (`testdata/fuzz/xxe/`).
- **Écriture atomique** (`internal/fsatomic`, temp + rename, ADR-010).
- **Lecteurs/écrivains** CII (D16B), UBL 2.1 (Invoice + CreditNote) et JSON pivot, via un
  registre ; **aller-retour lossless** vérifié pour CII et UBL (facture et avoir). Détection
  de format par le contenu (jamais l'extension seule).
- **Factur-X** : extraction du XML embarqué (`pkg/pdfa`, `pkg/read/facturx`, pdfcpu) et
  embarquement (`kanjo embed`). Boucle embed→extract testée.
- **Orchestration** (`pkg/convert`) : conversion via pivot + rapport de perte + politique
  `--max-loss`.
- **Moteur de validation** (`pkg/rules`) avec enveloppe de rapport ; sortie JSON versionnée
  (`pkg/api`, enveloppe `schemaVersion`).
- **Comparateur sémantique** (`pkg/diff`, `kanjo diff`) : compare deux pivots quels que soient
  leurs formats (Factur-X vs UBL), distingue **pertes** et **divergences** (§G5).
- **Pipeline de traitement par lot** (`pkg/pipeline`) : découverte (fichiers/dossiers/globs,
  récursif, filtres include/exclude), pool de workers borné, sûreté aux paniques (une panique
  devient une erreur de fichier, jamais un arrêt du lot), et **reprise `--resume`** (journal
  append-only : aucun fichier retraité, aucun perdu). Câblé dans `kanjo convert`
  (`--recursive`, `--workers`, `--include/--exclude`, `--resume`, `--fail-fast`).
- **Writers XRechnung** (syntaxes UBL et CII) et **Peppol BIS Billing 3.0** (UBL), via override
  du CustomizationID/ProfileID ; non-régression CII/UBL prouvée.
- **Presets** (`pkg/preset`, `kanjo preset list|show|save|delete|export|import`) : réglages de
  conversion nommés, stockés en JSON local (jamais de chemin absolu ni de secret, §G6).
  `kanjo convert --preset <nom>` (les flags explicites priment).
- **Surveillance de dossier** (`kanjo watch`) : scrutation pure Go (sans fsnotify), détection de
  fichier stable (taille inchangée sur 2 scans), dossiers `output/done/failed`, `.error.json` en
  quarantaine, mode `--once`, arrêt propre sur interruption. Affiche explicitement que la
  surveillance s'arrête à la fermeture (pas de service, §G7).
- **Writer CSV** (cible `csv`) : aplatissement dénormalisé du pivot (une ligne par ligne de facture).
- **`kanjo generate`** (`pkg/generate`) : corpus synthétique reproductible (scénarios simple,
  multi-tva, avoir, autoliquidation, intracommunautaire, acompte ; `--invalid`). Documents
  arithmétiquement conformes vérifiés par le moteur de règles.
- **`kanjo anonymize`** (`pkg/anonymize`) : remplacement déterministe des données personnelles
  (noms, adresses, SIREN, TVA, IBAN synthétique valide mod-97, contacts), totaux recalculés
  cohérents. Point de conformité RGPD (export sûr pour le support, §17.4).
- **`kanjo repair`** (`pkg/repair`) : corrections sûres uniquement (nettoyage d'identifiants,
  recalcul des totaux à partir de lignes cohérentes), journalisées avant/après, sauvegarde `.bak`.
  N'invente jamais de donnée métier (§8.5 MUST).
- **Validation par lot** : `kanjo validate` développe dossiers/globs (`--recursive`,
  `--include/--exclude`) et valide en parallèle (`--workers`).
- **Rendu de la face lisible** (`pkg/render`, `kanjo render --to html`) : facture HTML autonome
  (sans réseau) dans l'esprit du design 大福帳, formatage monétaire français (espace fine
  insécable U+202F, virgule décimale), sceau optionnel `--seal` reflétant un verdict réellement
  calculé (§17.7). Le rendu PDF passera par ce HTML et un moteur externe optionnel (§G10).
- **Rapport de validation HTML** (`kanjo validate --report rapport.html`, aussi `.json`) :
  rapport autonome imprimable groupé par règle (§G4), joignable à un dossier de conformité.
- **Interface texte (TUI)** (`kanjo tui`, ou `kanjo` sans argument sur un terminal) : bâtie sur
  Bubble Tea + Lipgloss (pur Go, `CGO_ENABLED=0` préservé). Chrome 勘定, validation d'un dossier
  via le pipeline, liste à sceaux (適/保/否) et volet de détails des anomalies, navigation
  clavier, revalidation. Consomme le même cœur que la CLI (une implémentation, plusieurs façades).
- **Couverture `pkg/model` portée à 85,7 %** (critère d'acceptation L1).
- **Catalogue de règles généré** (`docs/rules.md`) depuis le registre ; test de synchronisation
  qui échoue si le catalogue est désynchronisé (§8.2).
- **CLI** (`kanjo`) : `convert`, `extract`, `embed`, `inspect`, `diff`, `validate`, `doctor`,
  `version`, avec sortie `--format json` et codes de sortie normalisés (§11.4).
- Compilation `CGO_ENABLED=0` vérifiée sur les 6 cibles OS/arch (ADR-002).

### Ajouté (L3)
- **Bibliothèque locale** (`pkg/library`, `kanjo library index|list|forget|purge`) : index
  SQLite (`modernc.org/sqlite`, pur Go — `CGO_ENABLED=0` préservé, 6/6 cibles) des documents
  traités. Stocke des **métadonnées et empreintes, pas le contenu** (§17.4) ; recherche à
  facettes (texte, verdict, format, période), **droit à l'effacement** (`forget`) et **purge de
  rétention** (`purge --months`, défaut 13 (politique de rétention à définir)).
- **Journal d'audit** (`pkg/audit`, `kanjo audit list|export`) : journal horodaté append-only
  (JSONL, §17.5). **Aucune donnée personnelle** — uniquement horodatage, action, acteur,
  empreintes, formats, verdict, versions (test CI §17.4 qui échoue sur tout motif SIREN/IBAN/
  e-mail). `convert` et `validate` journalisent automatiquement chaque traitement. Export CSV/JSONL.

### Ajouté (L3 — fondation)
- **Kanjō Studio — serveur** (`kanjo studio`, `cmd/kanjo/studio`) : serveur HTTP local pur Go
  (ADR-005) exposant une **API JSON** (`/api/version`, `/api/formats`, `/api/validate`) et le
  **frontend embarqué** via `go:embed` (ADR-006). Sécurité §17.3 : liaison **127.0.0.1
  uniquement** par défaut (toute autre adresse exige `--i-understand`), **jeton de session**
  exigé sur `/api/*` et injecté dans la page servie, aucune sortie réseau. API : `/api/version`,
  `/api/formats`, `/api/validate`, `/api/inspect`, `/api/convert`. Testé via httptest.
- **Kanjō Studio — frontend** (`gui/frontend/dist`, vanilla JS + CSS écrit à la main, sans
  framework tiers ni chargement réseau, embarqué via `go:embed`) : **système de design 大福帳
  complet** (`styles/tokens.css` — deux thèmes 昼/夜, échelle typographique, filet de reliure,
  **sceau 朱印 avec animation d'apposition** et `prefers-reduced-motion`, aucune ombre ni rayon).
  Chrome (rail idéogrammes, barre de titre, barre d'état à compteurs de verdict), écrans
  **玄関 Accueil** (glisser-déposer), **検 Inspecteur** (termes BT + anomalies, sceau animé au
  verdict), **証 Rapport** (groupé par règle), bascule de thème persistée, **palette de commandes
  ⌘K**. Formatage monétaire français. Les écrans restants (流/蔵/型/番/記録/設定, éditeur G9) et
  le wrapper Wails restent à développer.

- **Kanjō Studio — client lourd desktop** (`cmd/kanjo-studio`) : application native (fenêtre
  WebView WebKit) qui embarque en interne le même serveur (127.0.0.1 + jeton) et sert le même
  frontend/API — une seule base de code UI, deux façades (ADR-005, §19.1). **Artefact séparé
  compilé avec cgo** ; isolé par build tags (`//go:build cgo` / `!cgo`) pour que le binaire
  `kanjo` (CLI/TUI/studio) reste pur Go et que `CGO_ENABLED=0 go build ./...` continue de passer.
  Empaqueté en bundle `.app` macOS. (Le wrapper Wails v3 formel — icônes, menus, notarisation
  multi-OS — reste une évolution possible.)

### Ajouté (complétude)
- **Unités mineures par devise** (`model.MinorUnits`) : la TVA s'arrondit à la précision de la
  devise (2 par défaut, **0 pour JPY/HUF/KRW…**, 3 pour KWD/BHD…). Corrige le dernier faux
  positif du corpus officiel → **33/34 conformes** en EN 16931 pur (le dernier utilise une
  catégorie de TVA hors EN 16931, correctement rejeté).
- **Remises et charges de niveau document (BG-20/21)** lues, écrites et préservées en CII et
  UBL (montant, base, pourcentage, motif, catégorie de TVA). Round-trip CII→UBL→CII vérifié
  lossless sur une facture réelle à remises/charges (37 termes identiques). Débloque
  **BR-CO-11** (total des remises) et **BR-CO-12** (total des charges) → jeu porté à **57 règles**.
- **Lecteur ZUGFeRD 1.0** (`pkg/read/zugferd1`) : format hérité `rsm:CrossIndustryDocument`
  (CII D14B), détecté et lu vers le pivot. Les vraies factures ZUGFeRD 1.0 (Kraxi GmbH,
  Bei Spiel GmbH) sont désormais lisibles et inspectables.
- **Identifiant de partie BT-29** lu et écrit en CII (`ram:ID`/`GlobalID`) et UBL
  (`cac:PartyIdentification`) ; **adresse électronique BT-34** en UBL via `cbc:EndpointID`.
- **5 règles EN 16931 supplémentaires** (jeu porté à **55**) : BR-08/BR-10 (adresses),
  BR-CO-26 (identification du vendeur), BR-47 (catégorie de ventilation de TVA),
  BR-DEC-24 (décimales du montant de ligne). Catalogue `docs/rules.md` régénéré.

### Ajouté (présentation & CI)
- **README professionnel** (badges, captures d'écran, matrice de formats, cas d'usage) et
  **site GitHub Pages** (`docs/index.html`) reprenant le design 大福帳 de l'application, avec
  captures réelles (Studio, CLI, TUI, facture rendue) générées en headless.
- **CI GitHub Actions** (`.github/workflows/ci.yml`) : gofmt, vet, tests, cross-compilation des
  6 cibles `CGO_ENABLED=0`, vérification du catalogue de règles, déploiement de la GitHub Page.
- **Corpus de test reproductible** (`testdata/corpus/fetch.sh`) : récupération des artefacts
  officiels EN 16931 (CEN) et de Factur-X réelles (mustangproject), avec empreintes SHA-256.
- **Idéogrammes remplacés par des icônes** dans toutes les façades (Studio : SVG style Lucide ;
  TUI : symboles ✓ ▲ ✕ ; rendu HTML : symboles) — les kanji ne subsistent qu'en identité de
  marque, doublés d'un libellé français et d'un `aria-label` (§12.10).

### Corrigé
- **Second `ram:TaxTotalAmount` (devise de comptabilisation, BT-111) en CII** : le lecteur ne
  retient que le montant dans la devise du document — corrige des faux BR-CO-14/15 sur les
  factures CII multi-devises (corpus CEN).
- **BR-CO-25 restreinte aux factures** : un avoir n'a pas d'échéance de paiement au sens du
  montant dû — aligné sur le validateur CEN. Résultat : **32/34 cas valides du corpus officiel
  EN 16931 conformes** en EN 16931 pur (les 2 restants : arrondi HUF, catégorie hors norme
  correctement rejetée).
- **Lecture du nom de partie en UBL** : le nom (BT-27/BT-44) est lu depuis
  `cac:PartyLegalEntity/cbc:RegistrationName` (avec repli sur `cac:PartyName`) — corrige des
  faux positifs BR-06/BR-07 sur de vraies factures UBL valides (corpus CEN : conformes 7 → 18).
- **Second `cac:TaxTotal` (devise de comptabilisation TVA, BT-111)** : le lecteur UBL ne retient
  désormais que le `TaxTotal` exprimé dans la devise du document — corrige des faux BR-CO-14/15
  sur les factures multi-devises.
- **Motif d'exonération (BT-120/121) perdu à l'aller-retour** : les lecteurs et écrivains CII et
  UBL ne portaient pas le motif d'exonération de TVA, rendant non conformes les factures en
  autoliquidation / intracommunautaire / export après conversion. Désormais préservé dans les
  deux syntaxes, verrouillé par un test d'aller-retour sur tous les scénarios.

### Conformité
- **jeu de règles 2026.3** — **50 règles** réparties en trois jeux (réellement calculées, §17.7),
  chacune avec un test passant et un test échouant ; catalogue généré dans `docs/rules.md` :
  - **EN 16931** (46) :
    - présence en-tête : BR-02/03/04/05/06/07/09/11/16 ;
    - présence ligne : BR-21/23/24/25/26/27 ;
    - listes de codes : BR-CL-01/03/17 ;
    - cohérence des totaux : BR-CO-09/10/13/14/15/16/17/18/25 ;
    - décimales (max 2) : BR-DEC-12/13/14/16/19/23 ;
    - TVA au taux normal : BR-S-01/05 ;
    - TVA par catégorie (taux nul + motif d'exonération) : BR-Z-05, BR-E-05/10, BR-AE-05/10,
      BR-K-05/10, BR-G-05/10, BR-O-05/10.
  - **CIUS française** (`cius.fr`, 2) : identification du vendeur (FR-CTC-01), format SIREN (FR-SIREN-01).
  - **Kanjō** (2) : échéance ≥ émission (KANJO-DATE-01), IBAN modulo 97 (KANJO-IBAN-01).
