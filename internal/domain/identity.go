package domain

import "strings"

func NormalizeIdentity(id, species, accession, batch string) (string, string, string, string) {
	return strings.TrimSpace(id), strings.Join(strings.Fields(species), " "), strings.ToUpper(strings.TrimSpace(accession)), strings.ToUpper(strings.TrimSpace(batch))
}
