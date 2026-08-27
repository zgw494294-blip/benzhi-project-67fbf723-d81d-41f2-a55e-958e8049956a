package domain

import "strings"

func NormalizeProtocol(protocol TreatmentProtocol) TreatmentProtocol {
	protocol.ProtocolID = strings.TrimSpace(protocol.ProtocolID)
	protocol.LightRegime = strings.Join(strings.Fields(protocol.LightRegime), " ")
	protocol.Substrate = strings.Join(strings.Fields(protocol.Substrate), " ")
	protocol.LockedAt = nil
	protocol.LockedRevision = 0
	protocol.ContentDigest = ""
	return protocol
}
