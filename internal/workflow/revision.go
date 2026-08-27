package workflow

import (
	"fmt"

	"seed-germination-workbench/internal/domain"
)

func ValidateRevision(trial *domain.Trial, expected int64) error {
	if expected != trial.Revision {
		return fmt.Errorf("版本冲突：当前 revision=%d", trial.Revision)
	}
	return nil
}
