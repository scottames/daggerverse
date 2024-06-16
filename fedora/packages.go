package main

import (
	"context"
	"fmt"
	"path/filepath"
)

const etcYumReposD = "/etc/yum.repos.d/"

// Repo represents a yum repository object
type Repo struct {
	Url      string
	FileName string
	Keep     bool
}

// WithReposFromUrls will add the content at each given url and install them
// on the Container image prior to package installation via
// WithPackagesInstalled. Optionally removing the repository afterward, prior
// to exporting the container.
func (f *Fedora) WithReposFromUrls(
	ctx context.Context,
	// urls of yum repository files to install (i.e. GitHub raw file)
	urls []string,
	// If true, the repository will not be removed on the generated Container
	// image
	keep bool,
) *Fedora {
	for _, r := range urls {
		repo := Repo{
			Url:      r,
			Keep:     keep,
			FileName: filepath.Base(r),
		}
		f.Repos = append(f.Repos, &repo)
	}

	return f
}

// WithPackageGroupsInstalled will install the given package groups
//
//	note: not supported on ostree-based images, will be skipped
func (f *Fedora) WithPackageGroupsInstalled(
	ctx context.Context,
	// list of package groups to be installed
	groups []string,
) *Fedora {
	f.PackageGroupsInstalled = append(f.PackageGroupsInstalled, groups...)

	return f
}

// WithPackageGroupsRemoved will remove the given package groups
//
//	note: not supported on ostree-based images, will be skipped
func (f *Fedora) WithPackageGroupsRemoved(
	ctx context.Context,
	// list of package groups to be installed
	groups []string,
) *Fedora {
	f.PackageGroupsInstalled = append(f.PackageGroupsInstalled, groups...)

	return f
}

// WithPackagesInstalled will install the given packages on the generated
// Container image.
func (f *Fedora) WithPackagesInstalled(
	ctx context.Context,
	// list of packages to be installed
	packages []string,
) *Fedora {
	f.PackagesInstalled = packages

	return f
}

// WithPackagesRemoved will remove the given packages on the generated Container
// image.
func (f *Fedora) WithPackagesRemoved(
	ctx context.Context,
	// list of packages to be removed on the generated Container image
	packages []string,
) *Fedora {
	f.PackagesRemoved = packages

	return f
}

type Swap struct {
	Remove  string
	Install string
}

// Remove spec and install spec in one transaction
// equivalent to:
// `dnf swap <remove> <install>`
// ostree-based:
// `rpm-ostree override remove <remove> --install <install>`
func (f *Fedora) WithPackagesSwapped(
	ctx context.Context,
	// package to remove
	remove string,
	// package to install
	install string,
) *Fedora {
	swap := Swap{Remove: remove, Install: install}
	f.PackagesSwapped = append(f.PackagesSwapped, swap)

	return f
}

// ctrWithReposInstalled will get the f.Repos urls and install them in the
// returned Container image.
func (f *Fedora) ctrWithReposInstalled(ctr *Container) (*Container, error) {
	if f.Repos == nil {
		return ctr, nil
	}

	for _, r := range f.Repos {
		contents, err := httpGet(r.Url)
		if err != nil {
			return nil, fmt.Errorf("error getting repo (%s) url: %w", r.FileName, err)
		}

		fileOpts := ContainerWithNewFileOpts{
			Contents:    string(contents),
			Permissions: 0644,
			Owner:       "root:root",
		}
		ctr = ctr.WithNewFile(fmt.Sprintf("%s/%s", etcYumReposD, r.FileName), fileOpts)
	}

	return ctr, nil
}

// ctrWithReposRemoved will remove f.Repos which are not marked to be kept
// in the generated Container image.
func (f *Fedora) ctrWithReposRemoved(ctr *Container) *Container {
	cmd := []string{"rm", "-f"}
	run := false
	for _, r := range f.Repos {
		if !r.Keep {
			run = true
			cmd = append(cmd, fmt.Sprintf("%s/%s", etcYumReposD, r.FileName))
		}
	}

	if run {
		return ctr.WithExec(cmd)
	}

	return ctr
}

// ctrWithPackagesInstalledAndRemoved executes installing and removing packages
// as specified by the Fedora object
func (f *Fedora) ctrWithPackagesInstalledAndRemoved(
	ctx context.Context,
	ctr *Container,
) (*Container, error) {
	ostreeBootable, _ := ctr.Label(ctx, "ostree.bootable")
	if ostreeBootable == "true" {
		return f.ctrWithPackagesInstalledAndRemovedRpmOstree(ctx, ctr), nil
	}

	return f.ctrWithPackagesInstalledAndRemovedDnf(ctx, ctr), nil
}

// ctrWithPackagesInstalledAndRemovedDnf executes installing and removing
// packages using `dnf` as specified by the Fedora object
func (f *Fedora) ctrWithPackagesInstalledAndRemovedDnf(
	ctx context.Context,
	ctr *Container,
) *Container {
	const dnf = "dnf"

	if len(f.PackageGroupsRemoved) > 0 {
		ctr = f.ctrWithExec(ctr,
			[]string{dnf, "-y", "group", "remove"},
			f.PackageGroupsRemoved...,
		)
	}

	// dnf swap only supports two packages so remove & install must be done in
	// separate transactions. https://bugzilla.redhat.com/show_bug.cgi?id=1934883
	if len(f.PackagesRemoved) > 0 {
		ctr = f.ctrWithExec(ctr,
			[]string{dnf, "-y", "remove"},
			f.PackagesRemoved...,
		)
	}

	packageGroupsInstalled := len(f.PackageGroupsInstalled) > 0
	packagesInstalled := len(f.PackagesInstalled) > 0
	if packageGroupsInstalled || packagesInstalled {
		ctr = f.ctrWithExec(ctr,
			[]string{dnf, "-y", "upgrade"},
		)
	}

	if packageGroupsInstalled {
		ctr = f.ctrWithExec(ctr,
			[]string{dnf, "-y", "group", "remove"},
			f.PackageGroupsInstalled...,
		)
	}

	if packagesInstalled {
		ctr = f.ctrWithExec(ctr,
			[]string{dnf, "-y", "install"},
			f.PackagesInstalled...,
		)
	}

	for _, swap := range f.PackagesSwapped {
		ctr = f.ctrWithExec(ctr,
			[]string{dnf, "-y", "swap"},
			swap.Remove, swap.Install,
		)
	}

	return ctr.WithExec([]string{dnf, "clean", "all"})
}

// ctrWithPackagesInstalledAndRemoved will return the given Container with
// packages installed and removed as defined by the Fedora object
func (f *Fedora) ctrWithPackagesInstalledAndRemovedRpmOstree(
	ctx context.Context,
	ctr *Container,
) *Container {
	removePackages := len(f.PackagesRemoved) > 0
	installPackages := len(f.PackagesInstalled) > 0

	cmd := []string{"rpm-ostree", "override", "remove"}
	// Doing both actions in one command allows for replacing required
	// packages with alternatives
	if installPackages && removePackages {
		cmd = append(cmd, f.PackagesRemoved...)
		for _, p := range f.PackagesInstalled {
			cmd = append(cmd, fmt.Sprintf("--install=%s", p))
		}

		return f.ctrWithExec(ctr, cmd)

	} else if removePackages {
		return f.ctrWithExec(ctr,
			cmd,
			f.PackagesRemoved...,
		)
	} else if installPackages {
		return f.ctrWithExec(ctr,
			[]string{"rpm-ostree", "install"},
			f.PackagesInstalled...,
		)
	}

	for _, swap := range f.PackagesSwapped {
		ctr = f.ctrWithExec(ctr, cmd, swap.Remove, "--install", swap.Install)
	}

	return ctr
}
