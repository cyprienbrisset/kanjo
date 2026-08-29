# Catalogue des règles de validation — Kanjō

> Fichier **généré** depuis le registre des règles (`pkg/rules`).
> Ne pas éditer à la main. Régénérer avec `KANJO_REGEN=1 go test ./pkg/rules/...`.

Version du jeu de règles : **2026.3**

## en16931 (94 règles)

| ID | Gravité | Termes | Message |
|----|---------|--------|---------|
| BR-02 | error | BT-1 | Le numéro de facture est obligatoire. |
| BR-03 | error | BT-2 | La date d'émission est obligatoire. |
| BR-04 | error | BT-3 | Le code type de facture est obligatoire. |
| BR-05 | error | BT-5 | Le code devise est obligatoire. |
| BR-06 | error | BT-27 | Le nom du vendeur est obligatoire. |
| BR-07 | error | BT-44 | Le nom de l'acheteur est obligatoire. |
| BR-08 | error | BG-5 | L'adresse postale du vendeur est obligatoire. |
| BR-09 | error | BT-40 | Le code pays du vendeur est obligatoire. |
| BR-10 | error | BG-8 | L'adresse postale de l'acheteur est obligatoire. |
| BR-11 | error | BT-55 | Le code pays de l'acheteur est obligatoire. |
| BR-12 | error | BT-106 | Le total des montants nets de ligne est obligatoire. |
| BR-13 | error | BT-109 | Le total hors TVA est obligatoire. |
| BR-14 | error | BT-112 | Le total TVA comprise est obligatoire. |
| BR-15 | error | BT-115 | Le net à payer est obligatoire. |
| BR-16 | error | BG-25 | Une facture doit comporter au moins une ligne. |
| BR-21 | error | BT-126 | Chaque ligne doit avoir un identifiant. |
| BR-23 | error | BT-130 | Chaque ligne doit indiquer une unité de mesure. |
| BR-24 | error | BT-131 | Chaque ligne doit porter un montant net. |
| BR-25 | error | BT-153 | Chaque ligne doit désigner un article (nom). |
| BR-26 | error | BT-146 | Chaque ligne doit porter un prix net d'article. |
| BR-27 | error | BT-146 | Le prix net d'une ligne ne doit pas être négatif. |
| BR-28 | error | BT-148 | Le prix brut d'une ligne ne doit pas être négatif. |
| BR-29 | error | BT-73, BT-74 | La fin de période ne doit pas précéder son début. |
| BR-30 | error | BT-134, BT-135 | La fin de période de ligne ne doit pas précéder son début. |
| BR-31 | error | BT-92 | Une remise de niveau document doit porter un montant. |
| BR-32 | error | BT-95 | Une remise de niveau document doit porter une catégorie de TVA. |
| BR-33 | error | BT-97 | Une remise de niveau document doit porter un motif ou un code motif. |
| BR-36 | error | BT-99 | Une charge de niveau document doit porter un montant. |
| BR-37 | error | BT-102 | Une charge de niveau document doit porter une catégorie de TVA. |
| BR-38 | error | BT-104 | Une charge de niveau document doit porter un motif ou un code motif. |
| BR-41 | error | BT-136 | Une remise de ligne doit porter un montant. |
| BR-42 | error | BT-139 | Une remise de ligne doit porter un motif ou un code motif. |
| BR-43 | error | BT-141 | Une charge de ligne doit porter un montant. |
| BR-44 | error | BT-144 | Une charge de ligne doit porter un motif ou un code motif. |
| BR-45 | error | BT-116 | Chaque ventilation de TVA doit porter un montant imposable. |
| BR-46 | error | BT-117 | Chaque ventilation de TVA doit porter un montant de TVA. |
| BR-47 | error | BT-118 | Chaque ventilation de TVA doit indiquer une catégorie. |
| BR-55 | error | BT-25 | Une référence de facture antérieure doit porter un identifiant. |
| BR-AE-01 | error | BT-118 | Une catégorie en autoliquidation employée impose une ventilation de TVA correspondante. |
| BR-AE-02 | error | BT-31, BT-48 | Une catégorie en autoliquidation exige les identifiants TVA du vendeur et de l'acheteur. |
| BR-AE-05 | error | BT-119, BT-152 | Une TVA en autoliquidation doit avoir un taux de zéro. |
| BR-AE-10 | error | BT-120, BT-121 | Une TVA en autoliquidation doit indiquer un motif d'exonération. |
| BR-CL-01 | error | BT-5 | Le code devise doit être un code ISO 4217 valide. |
| BR-CL-03 | error | BT-3 | Le code type de facture doit appartenir à la liste UNTDID 1001. |
| BR-CL-17 | error | BT-118, BT-151 | Le code catégorie de TVA doit appartenir à la liste UNTDID 5305. |
| BR-CO-04 | error | BT-151 | Chaque ligne doit porter une catégorie de TVA. |
| BR-CO-09 | error | BT-31 | Le n° de TVA du vendeur doit commencer par un préfixe pays à deux lettres. |
| BR-CO-10 | error | BT-106, BT-131 | La somme des montants nets de ligne doit être égale au total des lignes (BT-106). |
| BR-CO-11 | error | BT-107, BT-92 | Le total des remises (BT-107) doit être égal à la somme des remises de niveau document. |
| BR-CO-12 | error | BT-108, BT-99 | Le total des charges (BT-108) doit être égal à la somme des charges de niveau document. |
| BR-CO-13 | error | BT-106, BT-107, BT-108, BT-109 | Le total HT (BT-109) doit être égal au total des lignes moins les remises plus les charges. |
| BR-CO-14 | error | BT-110, BT-117 | Le total de TVA (BT-110) doit être égal à la somme des montants de TVA par catégorie. |
| BR-CO-15 | error | BT-109, BT-110, BT-112 | Le total TTC (BT-112) doit être égal au total HT plus le total de TVA. |
| BR-CO-16 | error | BT-112, BT-113, BT-114, BT-115 | Le net à payer (BT-115) doit être égal au total TTC moins l'acompte plus l'arrondi. |
| BR-CO-17 | error | BT-116, BT-117, BT-119 | Le montant de TVA par catégorie (BT-117) doit être égal à la base multipliée par le taux. |
| BR-CO-18 | error | BG-23 | Une facture doit comporter au moins une ventilation de TVA. |
| BR-CO-19 | error | BT-73, BT-74 | Une période de facturation doit indiquer une date de début ou de fin. |
| BR-CO-20 | error | BT-134, BT-135 | Une période de ligne doit indiquer une date de début ou de fin. |
| BR-CO-25 | error | BT-9, BT-20, BT-115 | Si un montant reste dû, une date d'échéance ou des conditions de paiement sont requises. |
| BR-CO-26 | error | BT-29 | Le vendeur doit être identifiable (identifiant, SIREN/SIRET ou n° de TVA). |
| BR-DEC-01 | error | BT-92 | Un montant de remise/charge ne doit pas avoir plus de deux décimales. |
| BR-DEC-05 | error | BT-99 | Un montant de remise/charge ne doit pas avoir plus de deux décimales. |
| BR-DEC-09 | error | BT-106 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-12 | error | BT-109 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-13 | error | BT-110 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-14 | error | BT-112 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-18 | error | BT-115 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-19 | error | BT-116 | Un montant de la ventilation de TVA ne doit pas avoir plus de deux décimales. |
| BR-DEC-20 | error | BT-117 | Un montant de la ventilation de TVA ne doit pas avoir plus de deux décimales. |
| BR-DEC-23 | error | BT-131 | Le montant net d'une ligne ne doit pas avoir plus de deux décimales. |
| BR-DEC-24 | error | BT-136 | Un montant de remise/charge de ligne ne doit pas avoir plus de deux décimales. |
| BR-DEC-27 | error | BT-141 | Un montant de remise/charge de ligne ne doit pas avoir plus de deux décimales. |
| BR-E-01 | error | BT-118 | Une catégorie exonérée employée impose une ventilation de TVA correspondante. |
| BR-E-02 | error | BT-31, BT-32 | Une catégorie exonérée exige un identifiant TVA ou fiscal du vendeur. |
| BR-E-05 | error | BT-119, BT-152 | Une TVA exonérée doit avoir un taux de zéro. |
| BR-E-10 | error | BT-120, BT-121 | Une TVA exonérée doit indiquer un motif d'exonération. |
| BR-G-01 | error | BT-118 | Une catégorie à l'export employée impose une ventilation de TVA correspondante. |
| BR-G-02 | error | BT-31, BT-32 | Une catégorie à l'export exige un identifiant TVA ou fiscal du vendeur. |
| BR-G-05 | error | BT-119, BT-152 | Une TVA à l'export doit avoir un taux de zéro. |
| BR-G-10 | error | BT-120, BT-121 | Une TVA à l'export doit indiquer un motif d'exonération. |
| BR-IC-01 | error | BT-118 | Une catégorie intracommunautaire employée impose une ventilation de TVA correspondante. |
| BR-IC-02 | error | BT-31, BT-48 | Une catégorie intracommunautaire exige les identifiants TVA du vendeur et de l'acheteur. |
| BR-IC-05 | error | BT-119, BT-152 | Une TVA intracommunautaire doit avoir un taux de zéro. |
| BR-IC-10 | error | BT-120, BT-121 | Une TVA intracommunautaire doit indiquer un motif d'exonération. |
| BR-O-01 | error | BT-118 | Une catégorie hors champ employée impose une ventilation de TVA correspondante. |
| BR-O-05 | error | BT-119, BT-152 | Une TVA hors champ doit avoir un taux de zéro. |
| BR-O-10 | error | BT-120, BT-121 | Une TVA hors champ doit indiquer un motif d'exonération. |
| BR-S-01 | error | BT-151, BT-118 | Une ligne au taux normal impose une ventilation de TVA de catégorie « S ». |
| BR-S-02 | error | BT-31, BT-32 | Une catégorie au taux normal exige un identifiant TVA ou fiscal du vendeur. |
| BR-S-05 | error | BT-152 | Le taux de TVA d'une ligne au taux normal doit être supérieur à zéro. |
| BR-S-10 | error | BT-120, BT-121 | Une ventilation au taux normal ne doit pas porter de motif d'exonération. |
| BR-Z-01 | error | BT-118 | Une catégorie à taux zéro employée impose une ventilation de TVA correspondante. |
| BR-Z-02 | error | BT-31, BT-32 | Une catégorie à taux zéro exige un identifiant TVA ou fiscal du vendeur. |
| BR-Z-05 | error | BT-119, BT-152 | Une TVA à taux zéro doit avoir un taux de zéro. |

## cius.fr (2 règles)

| ID | Gravité | Termes | Message |
|----|---------|--------|---------|
| FR-CTC-01 | error | BT-31, BT-30 | Un vendeur français doit être identifié par un n° de TVA ou un SIREN/SIRET. |
| FR-SIREN-01 | error | BT-30 | Un SIREN doit comporter exactement 9 chiffres. |

## kanjo (2 règles)

| ID | Gravité | Termes | Message |
|----|---------|--------|---------|
| KANJO-DATE-01 | warning | BT-9, BT-2 | La date d'échéance ne doit pas précéder la date d'émission. |
| KANJO-IBAN-01 | warning | BT-84 | L'IBAN doit satisfaire la vérification modulo 97 (ISO 13616). |

