package domain

func openDeviations(trial *Trial) int {
	count := 0
	for _, deviation := range trial.Deviations {
		if !deviation.Resolved {
			count++
		}
	}
	return count
}
