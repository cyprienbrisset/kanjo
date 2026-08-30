// Package all est le point d'assemblage des jeux de règles : il importe « à blanc » tous les
// sous-paquets de règles pour déclencher leur enregistrement (init). Les façades importent ce
// paquet une seule fois pour disposer de l'ensemble des règles.
package all

import (
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/cius/fr"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/en16931"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/kanjo"
	_ "github.com/cyprienbrisset/kanjo/pkg/rules/orderx"
)
