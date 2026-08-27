package store

import (
	"encoding/json"
	"os"
)

func writeJSONAtomic(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, encoded, 0644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
