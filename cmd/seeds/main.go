package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	// https://pkg.go.dev/github.com/clockworksoul/mediawiki#section-readme

	"github.com/clockworksoul/mediawiki"
	"github.com/divan/num2words"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"

	"dataminers/internal/constants"
	"dataminers/internal/filesearch"
	"dataminers/internal/models"
)

const USERNAME = "REDACT"
const PASSWORD = "REDACT"

type Seed struct {
	Name             string
	Planet           string
	Produces         []string
	Growth           int
	MaxHarvest       int
	Yield            float64
	HasStages        bool
	Stages           []string
	DefaultGiftLevel int
	SellValue        int
}

var PLANETS = map[string]string{
	// find . -name 'Location*Planet.asset.meta' -exec bash -c "echo {} && grep 'guid:' {}" \;
	"225aea078019c984eba31e63b3349aaa": "Lava Lakes",
	"735c091f19f233647b7727a1767a6bf4": "Desert Dune",
	"ccea1c148c96e6e42b0bfbedc05d4e8f": "Iceladus",
	"682677b8ad04adc46969377f57428541": "Grey Planet",
	"21a0d7b19f612b34f89a3e99a357d421": "Blue Reef",
	"ed6f62d11f329e14c842e602173b7bb5": "Utopia",
}

var PLANETS_BY_NAME = map[string]string{
	"Lava":        "Lava Lakes",
	"Desert":      "Desert Dunes",
	"Ice":         "Iceladus",
	"Grey Planet": "Grey Planet",
	"Ocean":       "Blue Reef",
	"Utopia":      "Viridis",
}

var fns = template.FuncMap{
	"neq": func(x, y interface{}) bool {
		return x != y
	},
	"eq": func(x, y interface{}) bool {
		return x == y
	},
	"sub": func(y, x int) int {
		return x - y
	},
	"num2words": func(num interface{}) string {
		half := ""
		switch num := num.(type) {
		case int:
			return num2words.Convert(num)
		case float64:
			if num != float64(int(num)) {
				half = " and a half"
			}
			return num2words.Convert(int(num)) + half
		}
		return ""
	},
}

func CreatePage(client *mediawiki.Client, title string, text string) error {
	resp, err := client.Edit().Bot(true).CreateOnly(true).
		Title(title).
		Text(text).
		Summary("Automated Page Creation (SwyytchBot)").
		Do(context.Background())
	if err != nil {
		return err
	}
	if resp.Edit.Result != "Success" {
		return fmt.Errorf("Error creating page: %s", resp.Edit.Result)
	}
	return nil
}

func itemNameToTitle(name string) string {
	return string(name[0]) + strings.ToLower(name[1:])
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	storeRegistry := filesearch.NewStoreItemRegistry(constants.ASSET_BASE_DIR)
	guidCache := filesearch.NewCachedGUIDSearch()
	client, err := mediawiki.New("https://lkg.wiki.gg/api.php", "SwyytchBot")
	if err != nil {
		panic(err)
	}
	resp, err := client.BotLogin(context.Background(), USERNAME, PASSWORD)
	if err != nil {
		panic(err)
	}
	if resp.BotLogin.Result != "Success" {
		panic(fmt.Errorf("Login failed: %s", resp.BotLogin.Result))
	}

	err = filepath.Walk(constants.ASSET_BASE_DIR, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".asset" {
			return nil
		}
		mono := models.Asset{}
		fd, err := os.Open(path)
		if err != nil {
			log.Error().Err(err).Str("Path", path).Msg("Error opening file")
			return nil
		}
		defer fd.Close()
		err = yaml.NewDecoder(fd).Decode(&mono)
		if err != nil {
			log.Error().Err(err).Str("Path", path).Msg("Error unmarshalling YAML")
		}
		if mono.MonoBehaviour.Store > 0 {
			storeRegistry.MaybeRegisterStoreItem(mono.MonoBehaviour)
		}
		if mono.MonoBehaviour.ItemCategory != "Seeds" {
			return nil
		}
		seed := Seed{
			Name:             itemNameToTitle(mono.MonoBehaviour.ItemName),
			Planet:           "",
			Produces:         []string{},
			Growth:           mono.MonoBehaviour.CropProductionGuide[0].ProduceDuration,
			MaxHarvest:       mono.MonoBehaviour.CropProductionGuide[0].MaxProductionCycles,
			Yield:            float64(mono.MonoBehaviour.CropProductionGuide[0].PickAmount) + mono.MonoBehaviour.CropProductionGuide[0].ExtraPickPercent,
			Stages:           []string{},
			DefaultGiftLevel: mono.MonoBehaviour.DefaultGiftLevel,
			SellValue:        mono.MonoBehaviour.SellValue,
		}

		metaFile := path + ".meta"
		metaFd, err := os.Open(metaFile)
		if err != nil {
			log.Error().Err(err).Str("Path", metaFile).Msg("Error opening file")
			return nil
		}
		defer metaFd.Close()
		meta := models.Meta{}
		err = yaml.NewDecoder(metaFd).Decode(&meta)
		if err != nil {
			log.Error().Err(err).Str("Path", metaFile).Str("ItemName", seed.Name).Msg("Error unmarshalling YAML")
		}

		// PLANET
		guid := meta.GUID
		storeItem := storeRegistry.GetStoreItem(guid)
		if storeItem.ActiveAtLocation.GUID != "" {
			seed.Planet = PLANETS[storeItem.ActiveAtLocation.GUID]
		} else if strings.HasSuffix(seed.Name, "mixed seeds") {
			seed.Planet = PLANETS_BY_NAME[strings.Split(seed.Name, " ")[0]]
		} else {
			log.Warn().Str("ItemName", seed.Name).Str("GUID", guid).Msg("No planet found for seed")
		}

		// PRODUCTS
		for _, product := range mono.MonoBehaviour.CropProductionGuide {
			if product.ProducesItem.ItemToDrop.GUID != "" {
				guid := product.ProducesItem.ItemToDrop.GUID
				product, err := filesearch.GetItemNameFromGUID(guidCache, guid)
				if err != nil {
					log.Error().Err(err).Str("ItemName", seed.Name).Str("GUID", guid).Msg("Error finding itemname")
					return nil
				}
				seed.Produces = append(seed.Produces, itemNameToTitle(product))
			} else if product.ProducesItem.LootTable.GUID != "" {
				guid := product.ProducesItem.LootTable.GUID
				file, err := guidCache.FindFileByGUID(constants.ASSET_BASE_DIR, guid)
				if err != nil {
					log.Error().Err(err).Str("ItemName", seed.Name).Str("GUID", guid).Msg("Error finding loottablefile")
					return nil
				}
				if file == "" {
					log.Error().Str("ItemName", seed.Name).Str("GUID", guid).Msg("Loot table file not found")
					return nil
				}
				lootTableFd, err := os.Open(file)
				if err != nil {
					log.Error().Err(err).Str("Path", file).Msg("Error opening file")
					return nil
				}
				defer lootTableFd.Close()
				lootTable := models.Asset{}
				err = yaml.NewDecoder(lootTableFd).Decode(&lootTable)
				if err != nil {
					log.Error().Err(err).Str("Path", file).Msg("Error unmarshalling YAML")
				}
				for _, item := range lootTable.MonoBehaviour.LootTable {
					guid := item.ItemToDrop.GUID
					product, err := filesearch.GetItemNameFromGUID(guidCache, guid)
					if err != nil {
						log.Error().Err(err).Str("ItemName", seed.Name).Str("GUID", guid).Msg("Error finding itemname")
						return nil
					}
					seed.Produces = append(seed.Produces, itemNameToTitle(product))
				}

			}
		}

		// STAGES
		stageCount := len(mono.MonoBehaviour.CropProductionGuide[0].StageSprites)
		for i := 0; stageCount > 0 && i < stageCount+1; i++ { // Plus CropSprite
			filename := fmt.Sprintf("%s_growth_%d.png", seed.Produces[0], i)
			seed.Stages = append(seed.Stages, filename)
		}
		if stageCount > 0 {
			seed.HasStages = true
		}

		t, err := template.New("seed.tmpl").Funcs(fns).ParseFiles("templates/seed.tmpl")
		if err != nil {
			panic(err)
		}
		buf := new(bytes.Buffer)

		err = t.Execute(buf, seed)
		if err != nil {
			log.Error().Err(err).Str("ItemName", seed.Name).Msg("Error executing template")
			return nil
		}
		log.Info().Str("ItemName", seed.Name).Msg("Creating page")
		err = CreatePage(client, seed.Name, buf.String())
		if err != nil {
			log.Error().Err(err).Str("ItemName", seed.Name).Msg("Error creating page")
			return nil
		}
		return nil

	})
	if err != nil {
		fmt.Println(err)
	}
}
