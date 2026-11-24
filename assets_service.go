package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// the assets that ship inside the binary, so a fresh install already has a sound and a
// layout to pick from. Named after main.go's `assets` embed, which is the frontend dist.
//
//go:embed all:assets
var builtinAssets embed.FS

const (
	assetsAudioDir  = "assets/audio"
	assetsLayoutDir = "assets/layout"
)

// AssetsService lists the assets available to the break settings: one flat folder per type,
// read from the embedded filesystem, so the frontend only asks for a list.
type AssetsService struct {
	fsys fs.FS
}

func NewAssetsService(fsys fs.FS) *AssetsService {
	return &AssetsService{fsys: fsys}
}

func (svc *AssetsService) ListAudio() ([]Asset, error) {
	return svc.list(assetsAudioDir, AssetTypeAudio)
}

func (svc *AssetsService) ListLayouts() ([]Asset, error) {
	return svc.list(assetsLayoutDir, AssetTypeLayout)
}

// list reads one folder, one level deep: a nested folder is not an asset.
// shortcut: the name is the file name because there is no manifest yet, read displayName
// from the asset's package.json once importing assets is supported.
func (svc *AssetsService) list(dir string, typ AssetType) ([]Asset, error) {
	entries, err := fs.ReadDir(svc.fsys, dir)

	if err != nil {
		return nil, fmt.Errorf("failed to read %s assets: %w", typ, err)
	}

	// never nil, so the frontend gets [] instead of null for an empty folder
	assets := make([]Asset, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

		// a dot file like .gitkeep has no name left
		if name == "" {
			continue
		}

		assets = append(assets, Asset{Name: name, Type: typ})
	}

	return assets, nil
}
