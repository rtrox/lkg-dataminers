package filesearch

import (
	"dataminers/internal/models"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type StoreItemRegistry struct {
	baseDir string
	Items   map[string]models.AssetMonoBehavior
}

func NewStoreItemRegistry(baseDir string) *StoreItemRegistry {
	return &StoreItemRegistry{
		baseDir: baseDir,
		Items:   make(map[string]models.AssetMonoBehavior),
	}
}

func (s *StoreItemRegistry) MaybeRegisterStoreItem(mono models.AssetMonoBehavior) {
	if mono.Store == 0 {
		return
	}
	if mono.ItemForSale.GUID == "" {
		return
	}
	if _, ok := s.Items[mono.ItemForSale.GUID]; ok {
		return
	}
	if mono.ActiveAtLocation.GUID == "" {
		return
	}
	s.Items[mono.ItemForSale.GUID] = mono
}

func (s *StoreItemRegistry) GetStoreItem(guid string) models.AssetMonoBehavior {
	item, ok := s.Items[guid]
	if ok {
		return item
	}
	found := false
	ret := models.AssetMonoBehavior{}
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) != ".asset" {
			return nil
		}
		fd, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fd.Close()
		mono := models.Asset{}
		err = yaml.NewDecoder(fd).Decode(&mono)
		if err != nil {
			return err
		}
		if mono.MonoBehaviour.ItemForSale.GUID == guid {
			ret = mono.MonoBehaviour
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return models.AssetMonoBehavior{}
	}
	if found {
		s.Items[guid] = ret
		return ret
	}
	return models.AssetMonoBehavior{}
}
