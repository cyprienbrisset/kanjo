// Package model définit le modèle pivot de Kanjō : une représentation des factures
// électroniques neutre vis-à-vis de la syntaxe (CII, UBL, Factur-X…), alignée sur la
// sémantique de la norme EN 16931 (Business Terms BT-xx et Business Groups BG-xx).
//
// Contrainte architecturale (cahier des charges §4.2, ADR-002) : ce paquet n'importe
// QUE la bibliothèque standard Go. Il ne connaît ni fichier, ni réseau, ni XML.
//
// Règles de modélisation clés (§5.2) :
//   - jamais de float64 pour un montant → type Amount (entier en unités mineures) ;
//   - jamais de time.Time pour une date de facture → type Date{Year, Month, Day} ;
//   - « absent » et « vide » sont distingués (pointeurs / booléens Present) ;
//   - les codes sont des types nommés avec Valid()/Label(), pas des string nues.
package model
