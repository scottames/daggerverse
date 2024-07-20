package main

import (
	"context"
	"dagger/fedora/internal/dagger"
)

// FileFromSource represents a File to be placed in the generated Container
// image at the Destination
type FileFromSource struct {
	Destination string
	Source      *dagger.File
}

// WithFile will upload the given File (file) at the given destination
func (f *Fedora) WithFile(
	ctx context.Context,
	// path in Container image to place the source file
	destination string,
	// file to be uploaded to the Container image
	file *dagger.File,
) *Fedora {
	fileFromSource := FileFromSource{Source: file, Destination: destination}
	f.Files = append(f.Files, &fileFromSource)

	return f
}
