package store

import "path/filepath"

func (s *Store) path(id string) string {
	return filepath.Join(s.Dir, id+".json")
}

func (s *Store) eventPath(id string) string {
	return filepath.Join(s.Dir, id+".events")
}

func (s *Store) idemPath(id string) string {
	return filepath.Join(s.Dir, id+".idem")
}
