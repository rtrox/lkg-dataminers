package main

import (
	"fmt"
	std_image "image"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/image/draw"
	"gopkg.in/yaml.v2"

	"dataminers/internal/constants"
	"dataminers/internal/filesearch"
	"dataminers/internal/images"
	"dataminers/internal/models"
)

var guidSearch *filesearch.CachedGUIDSearch

type Record struct {
	MName        string      `json:"m_Name"`
	ItemName     string      `json:"itemName"`
	ItemSprite   models.File `json:"itemSprite"`
	ItemCategory string      `json:"itemCategory"`
}

func formatNewFilename(rec models.AssetMonoBehavior) string {
	if rec.ItemName == "" {
		return rec.MName
	}
	ret := string(rec.ItemName[0]) + strings.ToLower(rec.ItemName[1:])
	return strings.Replace(ret, " ", "_", -1)
}

func ProcessSeedGrowthImages(outdir string, itemName string, spriteFile string) error {
	trimmed := strings.TrimSuffix(spriteFile, ".asset")
	img_idx := 1
	for {
		spriteFile := fmt.Sprintf("%s_%02d.asset", trimmed, img_idx)
		if _, err := os.Stat(spriteFile); os.IsNotExist(err) {
			break
		}
		outfile := fmt.Sprintf("%s_growth_%02d.png", strings.TrimSuffix(itemName, " Seeds"), img_idx)
		outfile = filepath.Join(outdir, outfile)
		err := processSpriteFile(outfile, spriteFile)
		if err != nil {
			return fmt.Errorf("Error processing sprite file: %w", err)
		}
		img_idx++

	}
	return nil
}

func processImage(outfile string, sprite models.AssetSprite, textureFile string) error {
	img, err := images.ReadImage(textureFile)
	if err != nil {
		return fmt.Errorf("Error reading image: %w", err)
	}
	textureHeight := img.Bounds().Dy()
	cropRect := std_image.Rect(
		sprite.MRect.X,
		textureHeight-sprite.MRect.Y-sprite.MRect.Height,
		sprite.MRect.X+sprite.MRect.Width,
		textureHeight-sprite.MRect.Y,
	)
	cropped, err := images.CropImage(img, cropRect)
	if err != nil {
		return fmt.Errorf("Error cropping image: %w", err)
	}
	scaled := images.ScaleImage(cropped, draw.NearestNeighbor, constants.SCALE_FACTOR)

	err = images.WriteImage(scaled, outfile)
	if err != nil {
		return fmt.Errorf("Error writing image: %w", err)
	}
	return nil
}

func processSpriteFile(outfile string, spriteFile string) error {
	fd2, err := os.Open(spriteFile)
	if err != nil {
		return fmt.Errorf("Error opening sprite file: %w", err)
	}
	defer fd2.Close()
	sprite := models.Asset{}
	err = yaml.NewDecoder(fd2).Decode(&sprite)
	if err != nil {
		return fmt.Errorf("Error unmarshalling Sprite YAML: %w", err)
	}
	textureGUID := sprite.Sprite.MRD.Texture.GUID
	textureFile, err := guidSearch.FindFileByGUID(constants.TEXTURE_BASE_DIR, textureGUID)
	if err != nil {
		return fmt.Errorf("Error finding texture file: %w", err)
	}
	if textureFile == "" {
		return fmt.Errorf("Texture file not found")
	}
	return processImage(outfile, sprite.Sprite, textureFile)
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	guidSearch = filesearch.NewCachedGUIDSearch()

	item_csv, err := os.Open("./items.csv")
	if err != nil {
		log.Error().Err(err).Msg("Error opening items.csv")
		return
	}
	defer item_csv.Close()

	fail_log, err := os.Create("./images.log")
	if err != nil {
		log.Error().Err(err).Msg("Error creating images.log")
		return
	}
	defer fail_log.Close()
	log := zerolog.New(os.Stderr).With().Timestamp().Logger()

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
		if mono.MonoBehaviour.ItemCategory == "" {
			return nil
		}
		spriteGuid := mono.MonoBehaviour.ItemSprite.GUID
		spriteFile, err := guidSearch.FindFileByGUID(constants.SPRITE_BASE_DIR, spriteGuid)
		if err != nil {
			log.Error().Err(err).Str("GUID", spriteGuid).Msg("Error finding sprite file")
			return nil
		}
		if spriteFile == "" {
			log.Error().Str("GUID", spriteGuid).Msg("Sprite file not found")
			return nil
		}
		outdir := filepath.Join("./output", mono.MonoBehaviour.ItemCategory)
		err = os.MkdirAll(outdir, 0755)
		if err != nil {
			return fmt.Errorf("Error creating output directory: %w", err)
		}

		filename := formatNewFilename(mono.MonoBehaviour) + ".png"

		outfile := filepath.Join(outdir, filename)

		err = processSpriteFile(outfile, spriteFile)
		if err != nil {
			log.Error().Err(err).Str("Path", path).Str("SpriteFile", spriteFile).Msg("Error processing sprite file")
			return nil
		}
		switch mono.MonoBehaviour.ItemCategory {
		case "Seeds":
			err = ProcessSeedGrowthImages(outdir, mono.MonoBehaviour.ItemName, spriteFile)
			if err != nil {
				log.Error().Err(err).Str("Path", path).Str("SpriteFile", spriteFile).Msg("Error processing seed growth images")
			}
		default:
			// Do nothing
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("Error walking MonoBehaviour directory")
		return
	}
}
