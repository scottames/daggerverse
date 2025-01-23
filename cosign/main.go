// Cosign container image signing in a Dagger module
package main

import (
	"context"
	"dagger/cosign/internal/dagger"
)

// Cosign represents the cosign Dagger module type
type Cosign struct{}

// Sign will run cosign from the image, as defined by the cosignImage
// parameter, to sign the given Container image digests
//
// Note: keyless signing not supported as-is
//
// See https://edu.chainguard.dev/open-source/sigstore/cosign/an-introduction-to-cosign/
func (f *Cosign) Sign(
	ctx context.Context,
	// Cosign private key
	privateKey dagger.Secret,
	// Cosign password
	password dagger.Secret,
	// registry username
	//+optional
	registryUsername *string,
	// name of the image
	//+optional
	registryPassword *dagger.Secret,
	// Docker config
	//+optional
	dockerConfig *dagger.File,
	// Cosign container image
	//+optional
	//+default="chainguard/cosign:latest"
	cosignImage *string,
	// Cosign container image user
	//+optional
	//+default="nonroot"
	cosignUser *string,
	// Container image digests to sign
	digests ...string,
) ([]string, error) {
	stdouts := []string{}
	for _, d := range digests {
		cmd := []string{"cosign", "sign", d, "--key", "env://COSIGN_PRIVATE_KEY"}
		stdout, err := f.exec(ctx, privateKey, password, registryUsername, registryPassword, dockerConfig, cosignImage, cosignUser, nil, cmd)
		if err != nil {
			return nil, err
		}
		stdouts = append(stdouts, stdout)
	}
	return stdouts, nil
}

// Attest will run cosign from the image, as defined by the cosignImage
// parameter, to attest the SBOM of the given Container image digest
//
// Note: keyless signing not supported as-is
//
// See https://edu.chainguard.dev/open-source/sigstore/cosign/how-to-sign-an-sbom-with-cosign/
func (f *Cosign) Attest(
	ctx context.Context,
	// Cosign private key
	privateKey dagger.Secret,
	// Cosign password
	password dagger.Secret,
	// registry username
	//+optional
	registryUsername *string,
	// name of the image
	//+optional
	registryPassword *dagger.Secret,
	// Docker config
	//+optional
	dockerConfig *dagger.File,
	// Cosign container image
	//+optional
	//+default="chainguard/cosign:latest"
	cosignImage *string,
	// Cosign container image user
	//+optional
	//+default="nonroot"
	cosignUser *string,
	// Container image digest to attest
	digest string,
	// SBOM file
	sbomFile *dagger.File,
	// SBOM type
	//+optional
	//+default="spdxjson"
	sbomType string,
) (string, error) {
	cmd := []string{"cosign", "attest", "--type", sbomType, "--predicate", "/home/nonroot/sbom.json", digest, "--key", "env://COSIGN_PRIVATE_KEY"}
	stdout, err := f.exec(ctx, privateKey, password, registryUsername, registryPassword, dockerConfig, cosignImage, cosignUser, sbomFile, cmd)
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func (f *Cosign) exec(
	ctx context.Context,
	// Cosign private key
	privateKey dagger.Secret,
	// Cosign password
	password dagger.Secret,
	// registry username
	//+optional
	registryUsername *string,
	// name of the image
	//+optional
	registryPassword *dagger.Secret,
	// Docker config
	//+optional
	dockerConfig *dagger.File,
	// Cosign container image
	//+optional
	//+default="chainguard/cosign:latest"
	cosignImage *string,
	// Cosign container image user
	//+optional
	//+default="nonroot"
	cosignUser *string,
	// SBOM file
	//+optional
	sbomFile *dagger.File,
	// Command to be executed
	cmd []string,
) (string, error) {
	if registryUsername != nil && registryPassword != nil {
		pwd, err := registryPassword.Plaintext(ctx)
		if err != nil {
			return "", err
		}

		cmd = append(
			cmd,
			"--registry-username",
			*registryUsername,
			"--registry-password",
			pwd,
		)
	}
	cosign := dag.
		Container().
		From(*cosignImage).
		WithUser(*cosignUser).
		WithEnvVariable("COSIGN_YES", "true").
		WithSecretVariable("COSIGN_PASSWORD", &password).
		WithSecretVariable("COSIGN_PRIVATE_KEY", &privateKey)

	if dockerConfig != nil {
		cosign = cosign.WithMountedFile(
			"/home/nonroot/.docker/config.json",
			dockerConfig,
			dagger.ContainerWithMountedFileOpts{Owner: *cosignUser})
	}

	if sbomFile != nil {
		cosign = cosign.WithMountedFile(
			"/home/nonroot/sbom.json",
			sbomFile,
			dagger.ContainerWithMountedFileOpts{Owner: *cosignUser})
	}

	return cosign.WithExec(cmd).Stdout(ctx)
}
