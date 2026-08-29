# ADR-011 — Kanjō est un outil autonome (arbitrage Q7)

- **Date** : 2026-08-29
- **Statut** : Accepté
- **Décideur** : Product Owner (arbitrage rapporté par l'équipe Dev)

## Contexte

La question Q7 du cahier des charges (§25), marquée « à trancher avant L1 », demandait si
Kanjō devait devenir la brique de conversion de une plateforme agréée (PA) ou rester un outil autonome. Ce choix
conditionne le niveau de stabilité exigé de l'API publique `pkg/`.

## Décision

**Kanjō est un outil autonome.** Il n'est pas, à ce stade, la brique de conversion intégrée de
une plateforme agréée (PA).

## Conséquences

- L'API `pkg/` n'a **pas** à être figée dès L1 pour un consommateur externe : nous pouvons
  itérer sur les signatures internes tant que la CLI et ses contrats de sortie JSON
  (`schemaVersion`, §11.3) restent stables pour les intégrateurs.
- Le **contrat stable exposé aux tiers** est donc la CLI + la sortie JSON versionnée, pas les
  paquets Go. C'est ce que nous garantissons vis-à-vis de l'extérieur.
- Les paquets `pkg/model` et `pkg/rules` restent malgré tout conçus proprement (zéro dépendance
  hors bibliothèque standard) : si une réintégration dans une plateforme agréée (PA) est décidée plus tard, la
  réutilisation reste possible sans réécriture. Voir point d'extension §21.5.
- Cette décision peut être révisée par un nouvel ADR si la stratégie produit évolue (Q8, §25).
