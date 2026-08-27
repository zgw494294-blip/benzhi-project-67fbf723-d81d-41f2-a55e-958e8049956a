package web

import "seed-germination-workbench/internal/domain"

func (input actionInput) protocol() domain.TreatmentProtocol {
	return domain.TreatmentProtocol{
		ProtocolID:                  input.ProtocolID,
		StratificationDays:          input.StratificationDays,
		TemperatureCelsius:          input.TemperatureCelsius,
		LightRegime:                 input.LightRegime,
		Substrate:                   input.Substrate,
		ObservationDays:             input.ObservationDays,
		GerminationThresholdPercent: input.GerminationThresholdPercent,
	}
}
