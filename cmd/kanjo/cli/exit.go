package cli

// Codes de sortie normalisés (cahier des charges §11.4).
const (
	ExitOK          = 0 // succès
	ExitInternal    = 1 // erreur interne inattendue
	ExitUsage       = 2 // erreur d'usage (flags invalides)
	ExitValidation  = 3 // validation échouée (≥ seuil)
	ExitUnreadable  = 4 // fichier illisible / format non reconnu
	ExitLossRefused = 5 // conversion dégradante refusée
	ExitCapability  = 6 // dépendance/capacité indisponible (XSD, Ghostscript, veraPDF)
	ExitInterrupted = 7 // interruption utilisateur
	ExitQuota       = 8 // quota / limite dépassée
)
