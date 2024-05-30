package filesearch

import (
	"dataminers/internal/constants"
	"dataminers/internal/models"
	"os"

	"gopkg.in/yaml.v2"
)

func GetItemNameFromGUID(guidCache *CachedGUIDSearch, guid string) (string, error) {
	file, err := guidCache.FindFileByGUID(constants.ASSET_BASE_DIR, guid)
	if err != nil {
		return "", err
	}
	if file == "" {
		return "", err
	}
	produceFd, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer produceFd.Close()
	produce := models.Asset{}
	err = yaml.NewDecoder(produceFd).Decode(&produce)
	if err != nil {
		return "", err
	}
	return produce.MonoBehaviour.ItemName, nil
}
