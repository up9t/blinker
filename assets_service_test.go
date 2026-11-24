package main

import (
	"testing"
	"testing/fstest"
)

func TestAssetsServiceList(t *testing.T) {
	svc := NewAssetsService(fstest.MapFS{
		"assets/audio/one.ogg":          {},
		"assets/audio/two.mp3":          {},
		"assets/audio/.gitkeep":         {},
		"assets/audio/nested/three.ogg": {},
		"assets/layout/default.html":    {},
		"other/ignored.ogg":             {},
	})

	audio, err := svc.ListAudio()

	if err != nil {
		t.Fatal(err)
	}

	expectedAudio := []Asset{
		{Name: "one", Type: AssetTypeAudio},
		{Name: "two", Type: AssetTypeAudio},
	}

	if len(audio) != len(expectedAudio) {
		t.Fatalf("expected %+v, got %+v", expectedAudio, audio)
	}

	for i := range expectedAudio {
		if audio[i] != expectedAudio[i] {
			t.Fatalf("expected %+v, got %+v", expectedAudio, audio)
		}
	}

	layouts, err := svc.ListLayouts()

	if err != nil {
		t.Fatal(err)
	}

	if len(layouts) != 1 || layouts[0] != (Asset{Name: "default", Type: AssetTypeLayout}) {
		t.Fatalf("expected the default layout, got %+v", layouts)
	}
}

func TestAssetsServiceMissingFolder(t *testing.T) {
	if _, err := NewAssetsService(fstest.MapFS{}).ListAudio(); err == nil {
		t.Fatal("expected an error for a missing folder")
	}
}

// the real embed, so a build that loses the assets folder fails here
func TestBuiltinAssets(t *testing.T) {
	svc := NewAssetsService(builtinAssets)

	audio, err := svc.ListAudio()

	if err != nil {
		t.Fatal(err)
	}

	if !containsAsset(audio, Asset{Name: "solemn", Type: AssetTypeAudio}) {
		t.Fatalf("expected the solemn audio, got %+v", audio)
	}

	layouts, err := svc.ListLayouts()

	if err != nil {
		t.Fatal(err)
	}

	if !containsAsset(layouts, Asset{Name: "default", Type: AssetTypeLayout}) {
		t.Fatalf("expected the default layout, got %+v", layouts)
	}
}

func containsAsset(assets []Asset, want Asset) bool {
	for _, asset := range assets {
		if asset == want {
			return true
		}
	}

	return false
}
