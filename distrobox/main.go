// A Dagger module meant to provide some use in working with Distrobox
// primarily for bootstrapping container images
//
// https://github.com/89luca89/distrobox
package main

import (
	"context"
	"fmt"
	"path"
	"regexp"
)

type Distrobox struct{}

// ContainerWithDistroboxClonedToTmp returns a Container with the Distrobox
// project cloned inside
func (d *Distrobox) ContainerWithDistroboxClonedToTmp(
	// container to use to clone the distrobox project
	//   note: `git` is expected to be the entrypoint!
	// +optional
	// +default="cgr.dev/chainguard/git:latest"
	from string,
	// desired path inside the container to clone the distrobox project
	// +optional
	// +default="/tmp"
	destination string,
) *Container {
	return dag.
		Container().
		From(from).
		WithExec([]string{
			"clone",
			"https://github.com/89luca89/distrobox.git",
			"--single-branch", fmt.Sprintf("%s/distrobox", destination),
		})
}

// ContainerWithFileFromUrl will exec curl to grab url content and place it at
// the given (or default) path
func (d *Distrobox) ContainerWithFileFromUrl(
	// container to use to execute curl
	//   note: `curl` is expected to be the entrypoint!
	// +optional
	// +default="cgr.dev/chainguard/curl:latest"
	from string,
	// url to grab the content from
	url string,
	//  path to place the file at
	// +optional
	// +default="/tmp"
	destination string,
	// name the file should be saved as
	// +optional
	// +default=""
	name string,
) *Container {
	if name == "" {
		name = path.Base(destination)
	}
	filePath := fmt.Sprintf("%s/%s", destination, name)

	return dag.
		Container().
		From(from).
		WithExec([]string{
			"-fsSL", "-o", filePath, url,
		})
}

// ContainerWithHostExec will grab the appropriate version of the host-spawn
// binary from GitHub releases as it associates with the downloaded distrobox
// release
func (d *Distrobox) HostSpawnFile(
	ctx context.Context,
	// container to use to execute curl to retrieve the host-spawn binary
	//   note: `curl` is expected to be the entrypoint!
	// +optional
	// +default="cgr.dev/chainguard/curl:latest"
	from string,
	// cpu architecture
	// +optional
	// +default="x86_64"
	arch string,
) (*File, error) {
	hostSpawnVersion, err := d.FindStringSubmatchInFile(
		ctx, d.HostExecFile(
			"cgr.dev/chainguard/git:latest", // HostExecFile uses git (not curl)
			"/tmp"), `host_spawn_version="(.*?)"`,
	)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf(
		"https://github.com/1player/host-spawn/releases/download/%s/host-spawn-%s",
		hostSpawnVersion,
		arch,
	)
	return d.
		ContainerWithFileFromUrl(
			from, url, "/tmp", "host-spawn",
		).
		File("/tmp/host-spawn"), nil
}

// HostExecFile returns the distrobox-host-exec File
func (d *Distrobox) HostExecFile(
	// container to use to clone the distrobox project
	//   note: `git` is expected to be the entrypoint!
	// +optional
	// +default="cgr.dev/chainguard/git:latest"
	from string,
	// desired path inside the container to clone the distrobox project
	// +optional
	// +default="/tmp"
	path string,
) *File {
	return d.
		ContainerWithDistroboxClonedToTmp(from, path).
		File(
			fmt.Sprintf("%s/distrobox/distrobox-host-exec", path),
		)
}

// FindStringSubmatchInFile returns the string matchings the given regex pattern
// in the provided File
func (d *Distrobox) FindStringSubmatchInFile(
	ctx context.Context,
	// file to match string in
	file *File,
	// regex to match against
	regex string,
) (string, error) {
	fileContents, err := file.Contents(ctx)
	if err != nil {
		return "", err
	}

	version, err := d.regexFindStringSubmatch(
		fileContents,
		regex,
	)
	if err != nil {
		return version, fmt.Errorf("unable to match '%s' in file", regex)
	}

	return version, nil
}

// regexFindStringSubmatch returns the string matched to the given regex
func (d *Distrobox) regexFindStringSubmatch(
	// text to match regex against
	text string,
	// regex to match against
	regex string,
) (string, error) {
	match := regexp.
		MustCompile(regex).
		FindStringSubmatch(text)

	if match == nil {
		return "", fmt.Errorf("no value found for regex: %s", regex)
	}

	return match[1], nil
}
