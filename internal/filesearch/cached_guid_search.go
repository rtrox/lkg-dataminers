package filesearch

import (
	"dataminers/internal/models"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type CachedGUIDSearch struct {
	guidMap map[string]string
}

func NewCachedGUIDSearch() *CachedGUIDSearch {
	return &CachedGUIDSearch{
		guidMap: make(map[string]string),
	}
}

func (c *CachedGUIDSearch) FindFileByGUID(baseDir string, guid string) (string, error) {
	if val, ok := c.guidMap[guid]; ok {
		return val, nil
	}
	ret := ""
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) != ".meta" {
			return nil
		}
		fd, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fd.Close()
		meta := models.Meta{}
		err = yaml.NewDecoder(fd).Decode(&meta)
		if err != nil {
			return err
		}
		if meta.GUID == guid {
			ret = strings.TrimSuffix(path, ".meta")
			return filepath.SkipAll
		}
		return nil
	})
	if ret != "" {
		c.guidMap[guid] = ret
		return ret, nil
	}
	if err != nil {
		return "", err
	}
	return "", nil
}
