package packagemanager

import (
	"fmt"
	"io/ioutil"

	"github.com/Masterminds/semver"
	"github.com/vercel/turborepo/cli/internal/fs"
	"gopkg.in/yaml.v3"
)

// PnpmWorkspaces is a representation of workspace package globs found
// in pnpm-workspace.yaml
type PnpmWorkspaces struct {
	Packages []string `yaml:"packages,omitempty"`
}

var nodejsPnpm = PackageManager{
	Name:       "nodejs-pnpm",
	Slug:       "pnpm",
	Command:    "pnpm",
	Specfile:   "package.json",
	Lockfile:   "pnpm-lock.yaml",
	PackageDir: "node_modules",
	// pnpm v7+ changed their handling of '--'. We no longer need to pass it to pass args to
	// the script being run, and in fact doing so will cause the '--' to be passed through verbatim,
	// potentially breaking scripts that aren't expecting it.
	// We are allowed to use nil here because ArgSeparator already has a type, so it's a typed nil,
	// This could just as easily be []string{}, but the style guide says to prefer
	// nil for empty slices.
	ArgSeparator: nil,

	getWorkspaceGlobs: func(rootpath fs.AbsolutePath) ([]string, error) {
		bytes, err := ioutil.ReadFile(rootpath.Join("pnpm-workspace.yaml").ToStringDuringMigration())
		if err != nil {
			return nil, fmt.Errorf("pnpm-workspace.yaml: %w", err)
		}
		var pnpmWorkspaces PnpmWorkspaces
		if err := yaml.Unmarshal(bytes, &pnpmWorkspaces); err != nil {
			return nil, fmt.Errorf("pnpm-workspace.yaml: %w", err)
		}

		if len(pnpmWorkspaces.Packages) == 0 {
			return nil, fmt.Errorf("pnpm-workspace.yaml: no packages found. Turborepo requires pnpm workspaces and thus packages to be defined in the root pnpm-workspace.yaml")
		}

		return pnpmWorkspaces.Packages, nil
	},

	getWorkspaceIgnores: func(pm PackageManager, rootpath fs.AbsolutePath) ([]string, error) {
		// Matches upstream values:
		// function: https://github.com/pnpm/pnpm/blob/d99daa902442e0c8ab945143ebaf5cdc691a91eb/packages/find-packages/src/index.ts#L27
		// key code: https://github.com/pnpm/pnpm/blob/d99daa902442e0c8ab945143ebaf5cdc691a91eb/packages/find-packages/src/index.ts#L30
		// call site: https://github.com/pnpm/pnpm/blob/d99daa902442e0c8ab945143ebaf5cdc691a91eb/packages/find-workspace-packages/src/index.ts#L32-L39
		return []string{
			"**/node_modules/**",
			"**/bower_components/**",
		}, nil
	},

	Matches: func(manager string, version string) (bool, error) {
		if manager != "pnpm" {
			return false, nil
		}

		v, err := semver.NewVersion(version)
		if err != nil {
			return false, fmt.Errorf("could not parse pnpm version: %w", err)
		}
		c, err := semver.NewConstraint(">=7.0.0")
		if err != nil {
			return false, fmt.Errorf("could not create constraint: %w", err)
		}

		return c.Check(v), nil
	},

	detect: func(projectDirectory fs.AbsolutePath, packageManager *PackageManager) (bool, error) {
		specfileExists := projectDirectory.Join(packageManager.Specfile).FileExists()
		lockfileExists := projectDirectory.Join(packageManager.Lockfile).FileExists()

		return (specfileExists && lockfileExists), nil
	},
}
