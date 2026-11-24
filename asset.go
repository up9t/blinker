package main

type AssetType = string

const (
	// a sound played when a break ends
	AssetTypeAudio AssetType = "audio"

	// the html shown by the break overlay
	AssetTypeLayout AssetType = "layout"
)

// json tags are what the binding generator uses for the frontend field names
type Asset struct {
	// the file name without its extension, unique inside its type
	Name string    `json:"name"`
	Type AssetType `json:"type"`
}
