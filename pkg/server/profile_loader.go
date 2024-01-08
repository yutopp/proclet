package server

import (
	"encoding/json"
	"os"

	"github.com/cockroachdb/errors"

	"github.com/yutopp/koya/pkg/domain"
)

type ProfileFromFile struct {
	Path string
}

func NewProfileFromFile(path string) *ProfileFromFile {
	return &ProfileFromFile{
		Path: path,
	}
}

func (p *ProfileFromFile) Load() (*domain.Profile, error) {
	r, err := os.Open(p.Path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var profile *domain.Profile
	if err := json.NewDecoder(r).Decode(&profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (p *ProfileFromFile) Save(profile *domain.Profile) error {
	w, err := os.Create(p.Path)
	if err != nil {
		return errors.Wrapf(err, "failed to create file: %s", p.Path)
	}
	defer w.Close()

	if err := json.NewEncoder(w).Encode(profile); err != nil {
		return err
	}

	return nil
}
