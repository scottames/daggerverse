package main

import (
	"context"
	"dagger/fedora/internal/dagger"
)

// DirectoryFromSource represents a Directory to be placed in the generated
// Container image at the Destination
type DirectoryFromSource struct {
	Source      *dagger.Directory
	Destination string
}

// WithDirectory will upload the given Directory (directory) at the given destination
func (f *Fedora) WithDirectory(
	ctx context.Context,
	// path in Container image to place the source directory
	destination string,
	// directory to be uploaded to the Container image
	directory *dagger.Directory,
) *Fedora {
	dir := DirectoryFromSource{Source: directory, Destination: destination}
	f.Directories = append(f.Directories, &dir)

	return f
}
