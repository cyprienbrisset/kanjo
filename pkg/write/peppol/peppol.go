// Package peppol écrit une facture au format Peppol BIS Billing 3.0 (syntaxe UBL), un CIUS
// d'EN 16931. On réutilise le writer UBL existant en forçant le CustomizationID et le
// ProfileID normatifs de Peppol.
package peppol

import (
	"github.com/cyprienbrisset/kanjo/pkg/model"
	"github.com/cyprienbrisset/kanjo/pkg/write"
	"github.com/cyprienbrisset/kanjo/pkg/write/ubl"
)

func init() { write.Register("peppol", Write) }

// Identifiants normatifs de Peppol BIS Billing 3.0.
const (
	customizationPeppolBIS30 = "urn:cen.eu:en16931:2017#compliant#urn:fdc:peppol.eu:2017:poacc:billing:3.0"
	profilePeppolBIS30       = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"
)

// Write sérialise le document pivot en Peppol BIS Billing 3.0 (UBL).
func Write(doc *model.Document, opts write.Options) ([]byte, error) {
	opts.CustomizationID = customizationPeppolBIS30
	opts.ProfileID = profilePeppolBIS30
	return ubl.Write(doc, opts)
}
