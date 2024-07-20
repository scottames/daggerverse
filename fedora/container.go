package main

import (
	"context"
	"dagger/fedora/internal/dagger"
	"fmt"
	"strings"
)

// ContainerAddress returns the string representation of the source
// container address
func (f *Fedora) ContainerAddress(
	registry string,
	org *string,
	variant string,
	suffix *string,
	tag string,
) string {
	s := ""
	if suffix != nil {
		s = fmt.Sprintf("-%s", *suffix)
	}

	const sourceStrWithOrg = "%s/%s/%s%s:%s" // registry/org/variant+suffix:tag
	const sourceStrWithoutOrg = "%s/%s%s:%s" // registry/variant+suffix:tag

	if org != nil {
		return fmt.Sprintf(sourceStrWithOrg,
			registry,
			*org,
			variant,
			s,
			tag,
		)
	}

	return fmt.Sprintf(sourceStrWithoutOrg,
		registry,
		variant,
		s,
		tag,
	)
}

// Container returns a Fedora container as a dagger.Container object
func (f *Fedora) Container(ctx context.Context) (*dagger.Container, error) {
	ctr, err := f.
		ContainerFrom(
			ctx,
			f.ContainerAddress(
				f.Registry,
				f.Org,
				f.Variant,
				f.Suffix,
				f.Tag,
			),
		)
	if err != nil {
		return nil, err
	}

	return ctr, nil
}

// ContainerFrom returns a Fedora container as a dagger.Container object
func (f *Fedora) ContainerFrom(
	ctx context.Context,
	// base container image to pull FROM
	from string,
) (*dagger.Container, error) {
	ctr := dag.
		Container().
		From(from)

	ctr = f.ctrWithDirectoriesInstalled(ctr)
	ctr = f.ctrWithFilesInstalled(ctr)

	if f.Repos != nil {
		var err error
		ctr, err = f.ctrWithReposInstalled(ctr)
		if err != nil {
			return nil, err
		}
	}

	if f.ExecScriptPre != nil {
		var err error
		ctr, err = f.ctrExecScripts(ctx, ctr, f.ExecScriptPre)
		if err != nil {
			return nil, err
		}
	}

	for _, cmd := range f.ExecPre {
		ctr = ctr.WithExec(cmd)
	}

	if f.PackagesInstalled != nil || f.PackagesRemoved != nil || f.PackagesSwapped != nil {
		var err error
		ctr, err = f.ctrWithPackagesInstalledAndRemoved(ctx, ctr)
		if err != nil {
			return nil, err
		}
	}

	ctr = f.ctrWithReposRemoved(ctr)

	if f.ExecScriptPost != nil {
		var err error
		ctr, err = f.ctrExecScripts(ctx, ctr, f.ExecScriptPost)
		if err != nil {
			return nil, err
		}
	}

	if f.ExecScriptPre != nil || f.ExecScriptPost != nil {
		scripts := append(f.ExecScriptPre, f.ExecScriptPost...)
		var err error
		ctr, err = f.ctrScriptsCleanup(ctx, ctr, scripts)
		if err != nil {
			return nil, err
		}
	}

	for _, cmd := range f.ExecPost {
		ctr = ctr.WithExec(cmd)
	}

	ctr = f.ctrWithLabels(ctr)

	return ctr, nil
}

// ContainerVersionFromLabel returns the label value for the image version,
// defined as 'version' OR 'org.opencontainers.image.version'
// and a possible error
func (f *Fedora) ContainerVersionFromLabel(
	ctx context.Context,
	// Container to use to determine the release version from
	// +optional
	ctr *dagger.Container,
) (string, error) {
	if ctr == nil {
		var err error
		ctr, err = f.Container(ctx)
		if err != nil {
			return "", err
		}
	}

	version, err := ctr.Label(ctx, "version")
	if err != nil || len(version) <= 0 {
		version, err = ctr.Label(ctx, "org.opencontainers.image.version")
		if err != nil || len(version) <= 0 {
			return "", fmt.Errorf(
				"unable to determine version from container labels",
			)
		}
	}

	return version, nil
}

// ContainerReleaseVersionFromLabel returns the label value for the image
// version, defined as 'version' OR 'org.opencontainers.image.version'
//   - if the version contains sub-versions & dates delimited by '.' they will
//     parsed out
func (f *Fedora) ContainerReleaseVersionFromLabel(
	ctx context.Context,
	// Container to use to determine the release version from
	// +optional
	ctr *dagger.Container,
) (string, error) {
	version, err := f.ContainerVersionFromLabel(ctx, ctr)
	if err != nil {
		return "", err
	}
	releaseVersion := strings.Split(version, ".")
	if len(releaseVersion) <= 0 {
		return "", fmt.Errorf(
			"unable to determine release version from base image version: %s",
			version,
		)
	}

	return releaseVersion[0], nil
}

// ctrWithDirectoriesInstalled returns a container type with the Fedora object
// with Directories installed
func (f *Fedora) ctrWithDirectoriesInstalled(ctr *dagger.Container) *dagger.Container {
	for _, d := range f.Directories {
		ctr = ctr.WithDirectory(d.Destination, d.Source)
	}

	return ctr
}

// ctrWithFilesInstalled returns a container type with the Fedora object
// with Files installed
func (f *Fedora) ctrWithFilesInstalled(ctr *dagger.Container) *dagger.Container {
	for _, d := range f.Files {
		ctr = ctr.WithFile(d.Destination, d.Source)
	}

	return ctr
}

// ctrWithLabels returns a container type with the Fedora object
// with Labels added
func (f *Fedora) ctrWithLabels(ctr *dagger.Container) *dagger.Container {
	for _, l := range f.Labels {
		ctr = ctr.WithLabel(l.Name, l.Value)
	}

	return ctr
}

// ctrWithExec wraps Container.WithExec allowing the command and args to be
// separated
func (f *Fedora) ctrWithExec(
	ctr *dagger.Container,
	// command to be executed
	command []string,
	// arguments to be passed to the given command
	args ...string,
) *dagger.Container {
	if args != nil {
		command = append(command, args...)
	}

	return ctr.WithExec(command)
}

// ctrExecScripts adds the given scripts (files) to the Fedora container
// and executes them. They will be removed from the final image as part
// or the ctrScriptsCleanup step
func (f *Fedora) ctrExecScripts(
	ctx context.Context,
	// Container image to run scripts against
	ctr *dagger.Container,
	// scripts (files) to be run
	scripts []*dagger.File,
) (*dagger.Container, error) {
	for _, script := range scripts {
		scriptName, err := script.Name(ctx)
		if err != nil {
			return nil, err
		}
		scriptTmp := fmt.Sprintf("/tmp/%s", scriptName)
		ctr = ctr.WithFile(scriptTmp, script).WithExec([]string{scriptTmp})
	}

	return ctr, nil
}

// ctrScriptsCleanup removes the Fedora container scripts (files) added as part
// of the ctrExecScripts step
func (f *Fedora) ctrScriptsCleanup(
	ctx context.Context,
	// Container image to run scripts against
	ctr *dagger.Container,
	// scripts (files) to be removed
	scripts []*dagger.File,
) (*dagger.Container, error) {
	filesToDelete := []string{}
	for _, script := range scripts {
		scriptName, err := script.Name(ctx)
		if err != nil {
			return nil, err
		}
		scriptTmp := fmt.Sprintf("/tmp/%s", scriptName)
		filesToDelete = append(filesToDelete, scriptTmp)
	}
	cmd := append([]string{"rm", "-f"}, filesToDelete...)
	return ctr.WithExec(cmd), nil
}
