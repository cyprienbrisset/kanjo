package version

import "testing"

func TestGet(t *testing.T) {
	info := Get()
	if info.Rules != Rules || info.Schema != Schema {
		t.Errorf("Get() n'expose pas les constantes : %+v", info)
	}
	if info.Tool != Tool || info.Commit != Commit || info.BuildDate != BuildDate {
		t.Errorf("Get() incohérent avec les variables de build : %+v", info)
	}
	// Les constantes de contrat ne doivent jamais être vides.
	if Rules == "" || Schema == "" {
		t.Error("Rules et Schema doivent être renseignés")
	}
}
