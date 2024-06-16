// Fedora container image
//
// A Dagger module for working with and generating a container image from the
// specified source Fedora image wrapping the dagger.Container type with several
// Fedora specific methods
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func New(
	ctx context.Context,
	// Container registry
	// +optional
	registry string,
	// Container registry organization
	// +optional
	org *string,
	// Variant or image name
	// +optional
	variant string,
	// Variant suffix string
	// e.g. main (as related to ublue-os images)
	// +optional
	suffix *string,
	// Tag or major release version
	// +optional
	tag string,
) *Fedora {
	now := time.Now()

	return setFedoraVersions(ctx, &Fedora{
		Tag:      tag,
		Org:      org,
		Registry: registry,
		Suffix:   suffix,
		Variant:  variant,
		Date:     strings.Replace(now.Format(time.DateOnly), "-", "", -1), // 20241031
	})
}

// setFedoraVersions sets the base image & release version information for the
// given Fedora object
func setFedoraVersions(ctx context.Context, fedora *Fedora) *Fedora {
	var err error
	fedora.BaseImage = fedora.
		ContainerAddress(
			fedora.Registry,
			fedora.Org,
			fedora.Variant,
			fedora.Suffix,
			fedora.Tag,
		)
	baseImageCtr := dag.Container().From(fedora.BaseImage)
	fedora.BaseImageVersion, _ = fedora.ContainerVersionFromLabel(ctx, baseImageCtr)
	releaseVersion, err := fedora.ContainerReleaseVersionFromLabel(ctx, baseImageCtr)
	// NOTE: major version skipped if not found! (not treated as error)
	if err == nil {
		fedora.ReleaseVersion = &releaseVersion
	}

	return fedora
}

// Fedora represents the constructed Fedora image
type Fedora struct {
	Org              *string
	Registry         string
	Suffix           *string
	Tag              string
	Variant          string
	ReleaseVersion   *string
	BaseImage        string
	BaseImageVersion string

	Date    string
	Digests []string

	Directories            []*DirectoryFromSource
	Files                  []*FileFromSource
	PackageGroupsInstalled []string
	PackageGroupsRemoved   []string
	PackagesInstalled      []string
	PackagesRemoved        []string
	PackagesSwapped        []Swap
	Repos                  []*Repo
	ExecScriptPre          []*File
	ExecScriptPost         []*File
	ExecPre                [][]string
	ExecPost               [][]string
	Labels                 []*ContainerLabel
}

func (f *Fedora) GetBaseImage() string {
	return f.BaseImageVersion
}

// httpGet will get the given url and return the data
func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error getting url '%s': %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status for url '%s': %v", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading data from url '%s': %w", url, err)
	}

	return data, nil
}

// DefaultTags returns the default image tags for the output Fedora image
// as:
//
//	<release version> (if determined),
//	<release version>-<date>,
//	<latest> (if set to true)
func (f *Fedora) DefaultTags(
	// if true the "latest" tag will be appended to the returned list
	// +optional
	// +default=false
	latest bool,
) []string {
	tags := []string{}
	if f.ReleaseVersion != nil {
		tags = append(tags,
			*f.ReleaseVersion,
			fmt.Sprintf("%s-%s", *f.ReleaseVersion, f.Date),
		)
	}
	if latest {
		tags = append(tags, "latest")
	}

	tags = append(tags, f.Date)

	return tags
}
