<div align="center">

# 勘定 Kanjō

**Lire · valider · convertir · réparer · comparer · rendre lisibles vos factures électroniques**
Factur-X · UBL 2.1 · CII · XRechnung · Peppol — une seule implémentation, trois interfaces.

[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![CGO disabled](https://img.shields.io/badge/CGO-disabled-5E7A4A)](#architecture)
[![OS](https://img.shields.io/badge/OS-Linux%20·%20macOS%20·%20Windows-24405E)](#démarrer)
[![Tests](https://img.shields.io/badge/tests-passing-5E7A4A)](#qualité)
[![EN 16931](https://img.shields.io/badge/EN%2016931-73%20règles-24405E)](#conformité)
[![RGPD](https://img.shields.io/badge/RGPD-100%25%20hors--ligne-9E2B32)](#sécurité--rgpd)
[![Façades](https://img.shields.io/badge/interfaces-CLI%20·%20TUI%20·%20Studio-B8862F)](#trois-interfaces-un-seul-cœur)
[![Licence](https://img.shields.io/badge/licence-à%20arbitrer-8E9AA6)](#licence)

📖 **[Site web / démonstration →](https://cyprienbrisset.github.io/kanjo/)** &nbsp;·&nbsp; [Changelog](CHANGELOG.md)

</div>

---

Depuis le **1ᵉʳ septembre 2026**, toutes les entreprises françaises assujetties à la TVA doivent
pouvoir **recevoir** des factures électroniques structurées. **Kanjō** (勘定, « le compte ») est
l'outil qui manipule ces objets : il les lit, les valide selon **EN 16931** et les CIUS françaises,
les convertit sans perte, les répare, les compare et les rend lisibles — du poste du comptable
jusqu'aux plateformes SaaS.

> **Principe fondateur** — un **modèle pivot** unique (sémantique EN 16931). Ajouter un format =
> 1 lecteur + 1 écrivain, jamais 2N convertisseurs. Le cœur ne dépend que de la bibliothèque
> standard Go ; **aucun appel réseau**, **aucune télémétrie**, traitement 100 % local.

<div align="center">
<img src="docs/assets/img/studio-inspector.png" alt="Kanjō Studio — Inspecteur" width="80%">
</div>

## Trois interfaces, un seul cœur

| | | |
|---|---|---|
| **CLI** | automatisation, intégration, CI | `kanjo convert · validate · extract · embed · diff · watch …` |
| **TUI** | inspection interactive au terminal | `kanjo tui ./factures` |
| **Studio** | application de bureau pour les comptables | `kanjo studio` (client lourd natif ou navigateur) |

<table>
<tr>
<td width="50%"><img src="docs/assets/img/cli.png" alt="CLI"><br><em>Ligne de commande — sortie JSON stable, codes de sortie normalisés.</em></td>
<td width="50%"><img src="docs/assets/img/tui.png" alt="TUI"><br><em>Interface texte — inspection sans quitter le clavier.</em></td>
</tr>
</table>

## Fonctionnalités

- **Lire &amp; inspecter** — Factur-X (PDF/A-3), UBL Invoice/CreditNote, CII D16B, JSON pivot. Détection **par le contenu**, jamais par l'extension.
- **Valider** — moteur EN 16931 **natif en Go** (pas de Schematron/JVM) : `BR`, `BR-CO`, `BR-CL`, `BR-DEC`, `BR-S/Z/E/AE/K/G/O`, CIUS française (SIREN, mentions CTC), règles maison (IBAN mod-97, dates). **Réellement calculé, jamais simulé.**
- **Convertir** — CII ⇄ UBL, Factur-X, XRechnung (UBL/CII), Peppol BIS 3.0, JSON, CSV — avec **rapport de perte explicite** et politique `--max-loss`.
- **Traiter en lot** — découverte récursive, pool de workers, **reprise `--resume`**, quarantaine, **surveillance de dossier** (`watch`).
- **Comparer** — `diff` sémantique entre deux factures quels que soient leurs formats (distingue **pertes** et **divergences**).
- **Rendre lisible** — facture **HTML autonome** et **rapport de validation** imprimable, dans le design de l'application.
- **RGPD** — `anonymize` (remplacement déterministe des données personnelles), **bibliothèque SQLite locale** (droit à l'effacement, rétention), **journal d'audit** horodaté sans donnée personnelle.
- **Extraire / embarquer** — `extract` le XML d'une Factur-X, `embed` un XML dans un PDF.
- **Générer** — `generate` un corpus synthétique (scénarios TVA variés, cas volontairement invalides).

## Démarrer

```bash
# Compilation — pur Go, aucune dépendance runtime, cross-compilable sur 6 cibles
CGO_ENABLED=0 go build -o kanjo ./cmd/kanjo

kanjo validate ./factures --recursive              # valider un dossier
kanjo convert ./factures --to ubl --out ./ubl -w 8 # convertir en lot
kanjo diff facture.pdf facture.ubl.xml             # prouver une conversion sans perte
kanjo studio                                        # application de bureau
kanjo tui ./factures                                # interface texte
```

Client lourd desktop (fenêtre native, WebKit) :

```bash
CGO_ENABLED=1 go build -o "Kanjō Studio" ./cmd/kanjo-studio
```

## Formats

| Format | Lecture | Écriture | Validation |
|---|:---:|:---:|:---:|
| Factur-X (PDF/A-3, tous profils) | ✅ | ✅ *(embed)* | ✅ |
| CII D16B | ✅ | ✅ | ✅ |
| UBL 2.1 (Invoice + CreditNote) | ✅ | ✅ | ✅ |
| XRechnung (UBL &amp; CII) | 🔜 | ✅ | ✅ |
| Peppol BIS Billing 3.0 | 🔜 | ✅ | ✅ |
| JSON pivot · CSV | ✅ / — | ✅ / ✅ | ✅ |
| ZUGFeRD 1.0 (CII D14B, hérité) | ✅ | 🗺️ | ✅ |
| Order-X · EDIFACT · FatturaPA | 🗺️ roadmap | 🗺️ | 🗺️ |

## Conformité

- **73 règles** réparties en trois jeux (`en16931`, `cius.fr`, `kanjo`), chacune avec un test passant et un test échouant ; catalogue généré dans [`docs/rules.md`](docs/rules.md).
- Testé sur le **corpus de test officiel EN 16931** (CEN) : **33/34 cas valides conformes** en EN 16931 pur — les 2 restants sont un arrondi en devise à 0 décimale (HUF) et un fichier utilisant une catégorie de TVA hors EN 16931 (correctement rejeté).
- Testé sur de **vraies Factur-X** (mustangproject) : lecture, validation conforme, conversion **sans perte** (`diff` = 0) et rendu lisible.
- Corpus reproductible : [`testdata/corpus/fetch.sh`](testdata/corpus/fetch.sh) (non vendoré, licences respectées).

## Architecture

Hexagonale, cœur sans I/O :

```
Façades   │ cmd/kanjo (CLI · TUI · studio)  ·  cmd/kanjo-studio (desktop)
Domaine   │ pkg/model (pivot)  ·  pkg/rules (validation)  ·  pkg/convert  ·  pkg/diff
Adapt.    │ pkg/read/*  ·  pkg/write/*  ·  pkg/pdfa  ·  pkg/render  ·  pkg/library  ·  pkg/audit
Sécurité  │ internal/xmlsafe (anti-XXE)  ·  internal/fsatomic (écriture atomique)
```

- `pkg/model` et `pkg/rules` **n'importent que la bibliothèque standard** — testables et réutilisables.
- **`CGO_ENABLED=0`** compile sur les 6 cibles (Linux/macOS/Windows × amd64/arm64). Seul le client lourd desktop utilise cgo.
- Dépendances directes minimales : `pdfcpu` (PDF), `bubbletea`/`lipgloss` (TUI), `modernc.org/sqlite` (bibliothèque) — **toutes pures Go**.

## Sécurité &amp; RGPD

- XML durci : entités externes désactivées (**XXE**), DOCTYPE refusé, bornes anti-« billion laughs ».
- **Aucun appel réseau** par défaut, **aucune télémétrie** ; Studio écoute sur `127.0.0.1` avec jeton de session.
- Journaux et audit **sans donnée personnelle** (chemins, empreintes, identifiants techniques uniquement) — vérifié en test.
- `anonymize` pour transmettre un cas client au support sans exporter de données personnelles.

## Qualité

- Tests unitaires, golden files, aller-retour **lossless**, propriétés (arrondis), fuzz XML/PDF, intégration CLI, bout-en-bout GUI.
- `gofmt` + `go vet` propres ; **aucune `panic`** dans un chemin de traitement (converti en erreur de fichier).

## Licence

À définir (interne / source-available / dual).

---

<div align="center">
Édité par <strong>Société Cliente</strong> · l'identité visuelle s'inspire du <em>大福帳 (daifukuchō)</em>, le grand livre des marchands d'Edo.
</div>
