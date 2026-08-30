# Catalogue des règles de validation — Kanjō

> Fichier **généré** depuis le registre des règles (`pkg/rules`).
> Ne pas éditer à la main. Régénérer avec `KANJO_REGEN=1 go test ./pkg/rules/...`.

Version du jeu de règles : **2026.3**

## en16931 (161 règles)

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
| BR-17 | error | BT-59 | Le nom du bénéficiaire est obligatoire si un bénéficiaire est indiqué. |
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
| BR-61 | error | BT-84 | Un paiement par virement impose un identifiant de compte. |
| BR-62 | error | BT-34 | Une adresse électronique doit porter un identifiant de schéma. |
| BR-63 | error | BT-49 | Une adresse électronique doit porter un identifiant de schéma. |
| BR-64 | error | BT-157 | Un identifiant normalisé d'article doit porter un identifiant de schéma. |
| BR-AE-01 | error | BT-118 | Une catégorie en autoliquidation employée impose une ventilation de TVA correspondante. |
| BR-AE-02 | error | BT-31, BT-48 | Une catégorie en autoliquidation exige les identifiants TVA du vendeur et de l'acheteur. |
| BR-AE-03 | error | BT-31, BT-48 | Une remise en autoliquidation exige les identifiants TVA du vendeur et de l'acheteur. |
| BR-AE-04 | error | BT-31, BT-48 | Une charge en autoliquidation exige les identifiants TVA du vendeur et de l'acheteur. |
| BR-AE-05 | error | BT-119, BT-152 | Une TVA en autoliquidation doit avoir un taux de zéro. |
| BR-AE-06 | error | BT-96, BT-103 | Le taux de TVA d'une remise en autoliquidation est invalide. |
| BR-AE-07 | error | BT-96, BT-103 | Le taux de TVA d'une charge en autoliquidation est invalide. |
| BR-AE-08 | error | BT-116 | La base d'imposition d'une ventilation en autoliquidation doit égaler la somme des montants de cette catégorie. |
| BR-AE-09 | error | BT-117 | Le montant de TVA d'une ventilation en autoliquidation doit être nul. |
| BR-AE-10 | error | BT-120, BT-121 | Une TVA en autoliquidation doit indiquer un motif d'exonération. |
| BR-CL-01 | error | BT-3 | Le code type de facture doit appartenir à la liste UNTDID 1001. |
| BR-CL-03 | error | BT-5 | La devise des montants doit être un code ISO 4217. |
| BR-CL-04 | error | BT-5 | Le code devise de la facture doit être un code ISO 4217. |
| BR-CL-14 | error | BT-40, BT-55, BT-80 | Les codes pays doivent appartenir à la liste ISO 3166-1. |
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
| BR-CO-21 | error | BT-97 | Une remise de niveau document doit porter un motif ou un code motif. |
| BR-CO-22 | error | BT-104 | Une charge de niveau document doit porter un motif ou un code motif. |
| BR-CO-23 | error | BT-139 | Une remise de ligne doit porter un motif ou un code motif. |
| BR-CO-24 | error | BT-144 | Une charge de ligne doit porter un motif ou un code motif. |
| BR-CO-25 | error | BT-9, BT-20, BT-115 | Si un montant reste dû, une date d'échéance ou des conditions de paiement sont requises. |
| BR-CO-26 | error | BT-29 | Le vendeur doit être identifiable (identifiant, SIREN/SIRET ou n° de TVA). |
| BR-DEC-01 | error | BT-92 | Un montant de remise/charge ne doit pas avoir plus de deux décimales. |
| BR-DEC-02 | error | BT-93 | La base d'une remise/charge ne doit pas avoir plus de deux décimales. |
| BR-DEC-05 | error | BT-99 | Un montant de remise/charge ne doit pas avoir plus de deux décimales. |
| BR-DEC-06 | error | BT-100 | La base d'une remise/charge ne doit pas avoir plus de deux décimales. |
| BR-DEC-09 | error | BT-106 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-10 | error | BT-107 | Un montant total ne doit pas avoir plus de deux décimales. |
| BR-DEC-11 | error | BT-108 | Un montant total ne doit pas avoir plus de deux décimales. |
| BR-DEC-12 | error | BT-109 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-13 | error | BT-110 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-14 | error | BT-112 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-16 | error | BT-113 | Un montant total ne doit pas avoir plus de deux décimales. |
| BR-DEC-17 | error | BT-114 | Un montant total ne doit pas avoir plus de deux décimales. |
| BR-DEC-18 | error | BT-115 | Un montant monétaire ne doit pas avoir plus de deux décimales. |
| BR-DEC-19 | error | BT-116 | Un montant de la ventilation de TVA ne doit pas avoir plus de deux décimales. |
| BR-DEC-20 | error | BT-117 | Un montant de la ventilation de TVA ne doit pas avoir plus de deux décimales. |
| BR-DEC-23 | error | BT-131 | Le montant net d'une ligne ne doit pas avoir plus de deux décimales. |
| BR-DEC-24 | error | BT-136 | Un montant de remise/charge de ligne ne doit pas avoir plus de deux décimales. |
| BR-DEC-25 | error | BT-137 | La base d'une remise/charge de ligne ne doit pas avoir plus de deux décimales. |
| BR-DEC-27 | error | BT-141 | Un montant de remise/charge de ligne ne doit pas avoir plus de deux décimales. |
| BR-DEC-28 | error | BT-142 | La base d'une remise/charge de ligne ne doit pas avoir plus de deux décimales. |
| BR-E-01 | error | BT-118 | Une catégorie exonérée employée impose une ventilation de TVA correspondante. |
| BR-E-02 | error | BT-31, BT-32 | Une catégorie exonérée exige un identifiant TVA ou fiscal du vendeur. |
| BR-E-03 | error | BT-31, BT-32 | Une remise exonérée exige un identifiant TVA/fiscal du vendeur. |
| BR-E-04 | error | BT-31, BT-32 | Une charge exonérée exige un identifiant TVA/fiscal du vendeur. |
| BR-E-05 | error | BT-119, BT-152 | Une TVA exonérée doit avoir un taux de zéro. |
| BR-E-06 | error | BT-96, BT-103 | Le taux de TVA d'une remise exonérée est invalide. |
| BR-E-07 | error | BT-96, BT-103 | Le taux de TVA d'une charge exonérée est invalide. |
| BR-E-08 | error | BT-116 | La base d'imposition d'une ventilation exonérée doit égaler la somme des montants de cette catégorie. |
| BR-E-09 | error | BT-117 | Le montant de TVA d'une ventilation exonérée doit être nul. |
| BR-E-10 | error | BT-120, BT-121 | Une TVA exonérée doit indiquer un motif d'exonération. |
| BR-G-01 | error | BT-118 | Une catégorie à l'export employée impose une ventilation de TVA correspondante. |
| BR-G-02 | error | BT-31, BT-32 | Une catégorie à l'export exige un identifiant TVA ou fiscal du vendeur. |
| BR-G-03 | error | BT-31, BT-32 | Une remise à l'export exige un identifiant TVA/fiscal du vendeur. |
| BR-G-04 | error | BT-31, BT-32 | Une charge à l'export exige un identifiant TVA/fiscal du vendeur. |
| BR-G-05 | error | BT-119, BT-152 | Une TVA à l'export doit avoir un taux de zéro. |
| BR-G-06 | error | BT-96, BT-103 | Le taux de TVA d'une remise à l'export est invalide. |
| BR-G-07 | error | BT-96, BT-103 | Le taux de TVA d'une charge à l'export est invalide. |
| BR-G-08 | error | BT-116 | La base d'imposition d'une ventilation à l'export doit égaler la somme des montants de cette catégorie. |
| BR-G-09 | error | BT-117 | Le montant de TVA d'une ventilation à l'export doit être nul. |
| BR-G-10 | error | BT-120, BT-121 | Une TVA à l'export doit indiquer un motif d'exonération. |
| BR-IC-01 | error | BT-118 | Une catégorie intracommunautaire employée impose une ventilation de TVA correspondante. |
| BR-IC-02 | error | BT-31, BT-48 | Une catégorie intracommunautaire exige les identifiants TVA du vendeur et de l'acheteur. |
| BR-IC-03 | error | BT-31, BT-48 | Une remise intracommunautaire exige les identifiants TVA du vendeur et de l'acheteur. |
| BR-IC-04 | error | BT-31, BT-48 | Une charge intracommunautaire exige les identifiants TVA du vendeur et de l'acheteur. |
| BR-IC-05 | error | BT-119, BT-152 | Une TVA intracommunautaire doit avoir un taux de zéro. |
| BR-IC-06 | error | BT-96, BT-103 | Le taux de TVA d'une remise intracommunautaire est invalide. |
| BR-IC-07 | error | BT-96, BT-103 | Le taux de TVA d'une charge intracommunautaire est invalide. |
| BR-IC-08 | error | BT-116 | La base d'imposition d'une ventilation intracommunautaire doit égaler la somme des montants de cette catégorie. |
| BR-IC-09 | error | BT-117 | Le montant de TVA d'une ventilation intracommunautaire doit être nul. |
| BR-IC-10 | error | BT-120, BT-121 | Une TVA intracommunautaire doit indiquer un motif d'exonération. |
| BR-IC-11 | error | BT-72, BG-14 | Une facture intracommunautaire doit indiquer une date de livraison ou une période. |
| BR-IC-12 | error | BT-80 | Une facture intracommunautaire doit indiquer le pays de livraison. |
| BR-O-01 | error | BT-118 | Une catégorie hors champ employée impose une ventilation de TVA correspondante. |
| BR-O-02 | error | BT-151, BT-31, BT-48 | Une catégorie « hors champ de la TVA » interdit les identifiants de TVA. |
| BR-O-03 | error | BT-95, BT-31, BT-48 | Une catégorie « hors champ de la TVA » interdit les identifiants de TVA. |
| BR-O-04 | error | BT-102, BT-31, BT-48 | Une catégorie « hors champ de la TVA » interdit les identifiants de TVA. |
| BR-O-05 | error | BT-119, BT-152 | Une TVA hors champ doit avoir un taux de zéro. |
| BR-O-08 | error | BT-116 | La base d'imposition d'une ventilation hors champ doit égaler la somme des montants de cette catégorie. |
| BR-O-09 | error | BT-117 | Le montant de TVA d'une ventilation hors champ doit être nul. |
| BR-O-10 | error | BT-120, BT-121 | Une TVA hors champ doit indiquer un motif d'exonération. |
| BR-O-11 | error | BT-118 | Une ventilation « hors champ » interdit toute autre ventilation de TVA. |
| BR-O-12 | error | BT-151 | Une ventilation « hors champ » interdit les lignes d'une autre catégorie. |
| BR-O-13 | error | BT-95 | Une ventilation « hors champ » interdit les remises/charges d'une autre catégorie. |
| BR-O-14 | error | BT-102 | Une ventilation « hors champ » interdit les remises/charges d'une autre catégorie. |
| BR-S-01 | error | BT-151, BT-118 | Une ligne au taux normal impose une ventilation de TVA de catégorie « S ». |
| BR-S-02 | error | BT-31, BT-32 | Une catégorie au taux normal exige un identifiant TVA ou fiscal du vendeur. |
| BR-S-03 | error | BT-31, BT-32 | Une remise au taux normal exige un identifiant TVA/fiscal du vendeur. |
| BR-S-04 | error | BT-31, BT-32 | Une charge au taux normal exige un identifiant TVA/fiscal du vendeur. |
| BR-S-05 | error | BT-152 | Le taux de TVA d'une ligne au taux normal doit être supérieur à zéro. |
| BR-S-06 | error | BT-96, BT-103 | Le taux de TVA d'une remise au taux normal est invalide. |
| BR-S-07 | error | BT-96, BT-103 | Le taux de TVA d'une charge au taux normal est invalide. |
| BR-S-08 | error | BT-116 | La base d'une ventilation au taux normal doit égaler la somme des montants de même catégorie et même taux. |
| BR-S-09 | error | BT-117 | Le montant de TVA au taux normal doit égaler la base multipliée par le taux. |
| BR-S-10 | error | BT-120, BT-121 | Une ventilation au taux normal ne doit pas porter de motif d'exonération. |
| BR-Z-01 | error | BT-118 | Une catégorie à taux zéro employée impose une ventilation de TVA correspondante. |
| BR-Z-02 | error | BT-31, BT-32 | Une catégorie à taux zéro exige un identifiant TVA ou fiscal du vendeur. |
| BR-Z-03 | error | BT-31, BT-32 | Une remise à taux zéro exige un identifiant TVA/fiscal du vendeur. |
| BR-Z-04 | error | BT-31, BT-32 | Une charge à taux zéro exige un identifiant TVA/fiscal du vendeur. |
| BR-Z-05 | error | BT-119, BT-152 | Une TVA à taux zéro doit avoir un taux de zéro. |
| BR-Z-06 | error | BT-96, BT-103 | Le taux de TVA d'une remise à taux zéro est invalide. |
| BR-Z-07 | error | BT-96, BT-103 | Le taux de TVA d'une charge à taux zéro est invalide. |
| BR-Z-08 | error | BT-116 | La base d'imposition d'une ventilation à taux zéro doit égaler la somme des montants de cette catégorie. |
| BR-Z-09 | error | BT-117 | Le montant de TVA d'une ventilation à taux zéro doit être nul. |
| BR-Z-10 | error | BT-120, BT-121 | Une ventilation à taux zéro ne doit pas porter de motif d'exonération. |

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

