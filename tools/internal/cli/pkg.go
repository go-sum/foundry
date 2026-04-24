package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/tabwriter"

	"github.com/go-sum/foundry/tools/internal/discover"
	"github.com/go-sum/foundry/tools/internal/github"
	"github.com/go-sum/foundry/tools/internal/gitops"
	"github.com/go-sum/foundry/tools/internal/version"
	"github.com/spf13/cobra"
)

// Config holds the global CLI configuration shared by all subcommands.
type Config struct {
	Owner    string
	DryRun   bool
	RepoRoot string
}

// NewPkgCmd returns the "pkg" subcommand group.
func NewPkgCmd(cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "Manage subtree-split packages in the foundry monorepo",
	}

	cmd.AddCommand(
		newPkgListCmd(cfg),
		newPkgStatusCmd(cfg),
		newPkgPushCmd(cfg),
		newPkgReleaseCmd(cfg),
		newPkgSyncCmd(cfg),
		newPkgDeployCmd(cfg),
	)

	return cmd
}

func newPkgListCmd(cfg *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all discovered packages under pkg/",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgs, err := discover.DiscoverPackages(cfg.RepoRoot)
			if err != nil {
				return err
			}

			starterGoMod := filepath.Join(cfg.RepoRoot, "starter", "go.mod")

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tVERSION\tMODULE\tPREFIX\tTYPE")

			// Top-level first, then nested.
			for _, p := range pkgs {
				if p.Nested {
					continue
				}
				ver, _ := version.ReadGoModVersion(starterGoMod, p.Module)
				if ver == "" {
					ver = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\ttop-level\n", p.Name, ver, p.Module, p.Prefix)
			}
			for _, p := range pkgs {
				if !p.Nested {
					continue
				}
				ver, _ := version.ReadGoModVersion(starterGoMod, p.Module)
				if ver == "" {
					ver = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\tnested\n", p.Name, ver, p.Module, p.Prefix)
			}

			return w.Flush()
		},
	}
}

func newPkgStatusCmd(cfg *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "status <name|all>",
		Short: "Compare local split SHA with the remote mirror",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := github.NewClientFromEnv(cfg.Owner)
			if err != nil {
				return err
			}
			token := os.Getenv("GITHUB_ACCESS_TOKEN")

			pkgs, err := discover.ResolvePackages(cfg.RepoRoot, args[0])
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tLOCAL\tREMOTE\tSTATUS")

			_ = token // token is used by PushGit, not needed for status reads

			for _, pkg := range pkgs {
				localSHA, err := gitops.SplitSubtree(cfg.RepoRoot, pkg.Prefix)
				if err != nil {
					return fmt.Errorf("split %s: %w", pkg.Name, err)
				}

				remoteSHA, err := client.GetRef(ctx, pkg.MirrorRepo, "heads/main")
				if err != nil {
					return fmt.Errorf("get remote ref for %s: %w", pkg.Name, err)
				}

				status := "in sync"
				if remoteSHA == "" {
					status = "not pushed"
				} else if localSHA != remoteSHA {
					status = "out of sync"
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					pkg.Name, shortSHA(localSHA), shortSHA(remoteSHA), status)
			}

			return w.Flush()
		},
	}
}

func newPkgPushCmd(cfg *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "push <name|all>",
		Short: "Subtree split and push a package to its mirror repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := github.NewClientFromEnv(cfg.Owner)
			if err != nil {
				return err
			}
			token := os.Getenv("GITHUB_ACCESS_TOKEN")

			pkgs, err := discover.ResolvePackages(cfg.RepoRoot, args[0])
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			for _, pkg := range pkgs {
				if err := client.EnsureRepoExists(ctx, pkg.MirrorRepo); err != nil {
					return err
				}

				fmt.Fprintf(os.Stderr, "Syncing %s to %s/%s@main\n", pkg.Prefix, cfg.Owner, pkg.MirrorRepo)

				sha, err := gitops.SplitSubtree(cfg.RepoRoot, pkg.Prefix)
				if err != nil {
					return fmt.Errorf("split %s: %w", pkg.Name, err)
				}
				fmt.Fprintf(os.Stderr, "  split SHA: %s\n", sha)

				remoteSHA, err := client.GetRef(ctx, pkg.MirrorRepo, "heads/main")
				if err != nil {
					return fmt.Errorf("get remote ref for %s: %w", pkg.Name, err)
				}
				if remoteSHA == sha {
					fmt.Fprintf(os.Stderr, "  already in sync, skipping\n")
					continue
				}

				if err := github.PushGit(cfg.RepoRoot, token, cfg.Owner, pkg.MirrorRepo, sha,
					[]string{"refs/heads/main"}, cfg.DryRun); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func newPkgReleaseCmd(cfg *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "release <name|all> [version]",
		Short: "Release a versioned package to its mirror repo",
		Long: `Release a package by subtree-splitting and pushing to its mirror repo.

If version is omitted, the patch version from starter/go.mod is auto-incremented.
If version is specified, it must be greater than the current version in starter/go.mod.
After a successful release, starter/go.mod is updated with the new version.

Use 'all' to release every discovered top-level package; explicit version is not allowed with 'all'.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrAll := args[0]
			explicit := ""
			if len(args) == 2 {
				if nameOrAll == "all" {
					return fmt.Errorf("explicit version cannot be used with 'all'")
				}
				explicit = args[1]
			}

			client, err := github.NewClientFromEnv(cfg.Owner)
			if err != nil {
				return err
			}
			token := os.Getenv("GITHUB_ACCESS_TOKEN")

			pkgs, err := discover.ResolvePackages(cfg.RepoRoot, nameOrAll)
			if err != nil {
				return err
			}

			starterGoMod := filepath.Join(cfg.RepoRoot, "starter", "go.mod")
			ctx := cmd.Context()

			for _, pkg := range pkgs {
				if _, _, err := releasePackage(ctx, cfg, client, token, pkg, starterGoMod, explicit); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// releasePackage releases a single package by subtree-splitting and pushing to its mirror repo.
// If explicitVersion is empty, the patch version is auto-incremented.
// Returns whether a release was made and the new version string.
func releasePackage(ctx context.Context, cfg *Config, client github.Client, token string, pkg discover.Package, starterGoMod, explicitVersion string) (bool, string, error) {
	if err := client.EnsureRepoExists(ctx, pkg.MirrorRepo); err != nil {
		return false, "", err
	}

	sha, err := gitops.SplitSubtree(cfg.RepoRoot, pkg.Prefix)
	if err != nil {
		return false, "", err
	}
	fmt.Fprintf(os.Stderr, "  split SHA: %s\n", sha)

	// When no explicit version is given, skip if nothing changed since the last release.
	if explicitVersion == "" {
		currentVersion, err := version.ReadGoModVersion(starterGoMod, pkg.Module)
		if err != nil {
			return false, "", err
		}
		remoteTagSHA, err := client.GetRef(ctx, pkg.MirrorRepo, "refs/tags/"+currentVersion)
		if err != nil {
			return false, "", err
		}
		if remoteTagSHA == sha {
			fmt.Fprintf(os.Stderr, "No changes since %s, skipping %s\n", currentVersion, pkg.Name)
			return false, currentVersion, nil
		}
	}

	newVersion, err := resolveReleaseVersion(starterGoMod, pkg.Module, explicitVersion)
	if err != nil {
		return false, "", err
	}

	fmt.Fprintf(os.Stderr, "Releasing %s to %s/%s as %s\n", pkg.Prefix, cfg.Owner, pkg.MirrorRepo, newVersion)

	refs := []string{"refs/heads/main", "refs/tags/" + newVersion}
	if err := github.PushGit(cfg.RepoRoot, token, cfg.Owner, pkg.MirrorRepo, sha, refs, cfg.DryRun); err != nil {
		return false, "", err
	}

	if err := client.CreateRelease(ctx, pkg.MirrorRepo, newVersion, pkg.Prefix); err != nil {
		return false, "", err
	}

	if !cfg.DryRun {
		if err := version.WriteGoModVersion(starterGoMod, pkg.Module, newVersion); err != nil {
			return false, "", fmt.Errorf("update starter/go.mod: %w", err)
		}
		fmt.Fprintf(os.Stderr, "  updated %s to %s in starter/go.mod\n", pkg.Module, newVersion)
	} else {
		fmt.Fprintf(os.Stderr, "  [dry-run] would update %s to %s in starter/go.mod\n", pkg.Module, newVersion)
	}

	return true, newVersion, nil
}

// resolveReleaseVersion determines the version to release.
// If explicit is non-empty, validates it is > current. Otherwise, bumps patch.
func resolveReleaseVersion(starterGoMod, modulePath, explicit string) (string, error) {
	currentStr, err := version.ReadGoModVersion(starterGoMod, modulePath)
	if err != nil {
		return "", err
	}

	current, err := version.Parse(currentStr)
	if err != nil {
		return "", fmt.Errorf("current version in starter/go.mod: %w", err)
	}

	if explicit == "" {
		next := current.BumpPatch()
		return next.String(), nil
	}

	next, err := version.Parse(explicit)
	if err != nil {
		return "", err
	}

	if !next.GreaterThan(current) {
		return "", fmt.Errorf("version %s must be greater than current %s in starter/go.mod", next, current)
	}

	return next.String(), nil
}

func newPkgSyncCmd(cfg *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Copy starter/go.mod to starter/go.prod.mod with replace directives removed",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			starterGoMod := filepath.Join(cfg.RepoRoot, "starter", "go.mod")
			starterProdMod := filepath.Join(cfg.RepoRoot, "starter", "go.prod.mod")
			return syncProdMod(cfg.RepoRoot, starterGoMod, starterProdMod)
		},
	}
}

// syncProdMod regenerates go.prod.mod from go.mod (stripping replace directives)
// and runs go mod tidy to verify all dependencies resolve.
func syncProdMod(repoRoot, starterGoMod, starterProdMod string) error {
	if err := version.CopyModStripReplace(starterGoMod, starterProdMod); err != nil {
		return fmt.Errorf("write go.prod.mod: %w", err)
	}
	fmt.Fprintln(os.Stderr, "wrote starter/go.prod.mod (replace directives removed)")

	starterDir := filepath.Join(repoRoot, "starter")
	tidy := exec.Command("go", "mod", "tidy", "-modfile=go.prod.mod")
	tidy.Dir = starterDir
	tidy.Stdout = os.Stderr
	tidy.Stderr = os.Stderr
	tidy.Env = append(os.Environ(),
		"GOWORK=off",
		"GOPRIVATE=github.com/go-sum/*",
	)

	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy -modfile=go.prod.mod: %w", err)
	}
	fmt.Fprintln(os.Stderr, "starter/go.prod.mod tidied successfully")
	return nil
}

// packageCheck holds the validation result for a single package.
type packageCheck struct {
	pkg          discover.Package
	localSHA     string
	remoteSHA    string
	version      string
	needsRelease bool
}

func newPkgDeployCmd(cfg *Config) *cobra.Command {
	var autoFix bool

	cmd := &cobra.Command{
		Use:   "deploy [version]",
		Short: "Validate dependencies and optionally release, tag, and push",
		Long: `Checks that all packages in starter/go.mod are published and up to date.

Without --auto, reports dependency status and exits.
With --auto, releases stale packages, syncs starter/go.prod.mod, bumps APP_VERSION,
commits, tags, and pushes.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			explicit := ""
			if len(args) == 1 {
				explicit = args[0]
			}

			starterGoMod := filepath.Join(cfg.RepoRoot, "starter", "go.mod")
			starterProdMod := filepath.Join(cfg.RepoRoot, "starter", "go.prod.mod")
			starterProdSum := filepath.Join(cfg.RepoRoot, "starter", "go.prod.sum")

			// Preflight: reset deploy artifacts, then check clean tree and branch.
			if autoFix {
				deployArtifacts := []string{".versions", starterGoMod, starterProdMod, starterProdSum}
				_ = gitops.CheckoutHead(cfg.RepoRoot, deployArtifacts...)
			}
			if err := gitops.EnsureCleanTree(cfg.RepoRoot); err != nil {
				return err
			}
			if err := gitops.EnsureOnBranch(cfg.RepoRoot, "main"); err != nil {
				return err
			}

			currentVersion, nextVersion, err := resolveAppVersion(cfg.RepoRoot, explicit)
			if err != nil {
				return err
			}

			client, err := github.NewClientFromEnv(cfg.Owner)
			if err != nil {
				return err
			}
			token := os.Getenv("GITHUB_ACCESS_TOKEN")

			fmt.Fprintf(os.Stderr, "Deploy: %s → %s\n\n", currentVersion, nextVersion)

			pkgs, err := discover.ResolveTopLevel(cfg.RepoRoot)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			checks, err := validatePackages(ctx, cfg, client, pkgs, starterGoMod)
			if err != nil {
				return err
			}

			stale := countStale(checks)

			// Print validation table.
			printCheckTable(cmd, checks)
			fmt.Fprintln(os.Stderr)

			if !autoFix {
				if stale > 0 {
					fmt.Fprintf(os.Stderr, "%d package(s) need release. Run with --auto to fix, or release manually.\n", stale)
					return fmt.Errorf("%d stale package(s)", stale)
				}
				fmt.Fprintf(os.Stderr, "All dependencies valid. Ready to deploy %s.\n", nextVersion)
				fmt.Fprintln(os.Stderr, "Run with --auto to tag and push.")
				return nil
			}

			// Auto mode: release stale → sync → commit → tag → push.
			if stale > 0 {
				fmt.Fprintf(os.Stderr, "Releasing %d stale package(s)...\n\n", stale)
				for _, chk := range checks {
					if !chk.needsRelease {
						continue
					}
					released, newVer, err := releasePackage(ctx, cfg, client, token, chk.pkg, starterGoMod, "")
					if err != nil {
						return fmt.Errorf("release %s: %w", chk.pkg.Name, err)
					}
					if released {
						fmt.Fprintf(os.Stderr, "  %s: %s → %s\n", chk.pkg.Name, chk.version, newVer)
					}
				}
				fmt.Fprintln(os.Stderr)
			}

			fmt.Fprintln(os.Stderr, "Syncing starter/go.prod.mod...")
			if !cfg.DryRun {
				if err := syncProdMod(cfg.RepoRoot, starterGoMod, starterProdMod); err != nil {
					return fmt.Errorf("sync starter/go.prod.mod: %w", err)
				}
			} else {
				fmt.Fprintln(os.Stderr, "  [dry-run] would regenerate starter/go.prod.mod and run go mod tidy")
			}

			if !cfg.DryRun {
				if err := version.WriteDotVersion(cfg.RepoRoot, "APP_VERSION", nextVersion); err != nil {
					return fmt.Errorf("update APP_VERSION: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Updated APP_VERSION to %s\n", nextVersion)
			} else {
				fmt.Fprintf(os.Stderr, "[dry-run] would update APP_VERSION to %s\n", nextVersion)
			}

			commitMsg := fmt.Sprintf("chore: release %s", nextVersion)

			if !cfg.DryRun {
				if err := gitops.Add(cfg.RepoRoot, ".versions", starterGoMod, starterProdMod, starterProdSum); err != nil {
					return fmt.Errorf("git add: %w", err)
				}
				if err := gitops.Commit(cfg.RepoRoot, commitMsg); err != nil {
					return fmt.Errorf("git commit: %w", err)
				}
				if err := gitops.Tag(cfg.RepoRoot, nextVersion); err != nil {
					return fmt.Errorf("git tag: %w", err)
				}
				if err := gitops.PushWithTags(cfg.RepoRoot, "origin", "main"); err != nil {
					return fmt.Errorf("git push: %w", err)
				}
				fmt.Fprintf(os.Stderr, "\nTagged %s — CI build triggered\n", nextVersion)
			} else {
				fmt.Fprintf(os.Stderr, "\n[dry-run] would commit, tag %s, and push to origin\n", nextVersion)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&autoFix, "auto", false, "auto-release stale packages, sync, tag, and push")
	return cmd
}

// validatePackages checks each package to determine if it needs release (read-only).
func validatePackages(ctx context.Context, cfg *Config, client github.Client, pkgs []discover.Package, starterGoMod string) ([]packageCheck, error) {
	var checks []packageCheck

	for _, pkg := range pkgs {
		sha, err := gitops.SplitSubtree(cfg.RepoRoot, pkg.Prefix)
		if err != nil {
			return nil, fmt.Errorf("split %s: %w", pkg.Name, err)
		}

		ver, err := version.ReadGoModVersion(starterGoMod, pkg.Module)
		if err != nil {
			return nil, fmt.Errorf("read version for %s: %w", pkg.Name, err)
		}

		remoteSHA, err := client.GetRef(ctx, pkg.MirrorRepo, "refs/tags/"+ver)
		if err != nil {
			return nil, fmt.Errorf("get remote ref for %s: %w", pkg.Name, err)
		}

		checks = append(checks, packageCheck{
			pkg:          pkg,
			localSHA:     sha,
			remoteSHA:    remoteSHA,
			version:      ver,
			needsRelease: sha != remoteSHA,
		})
	}

	return checks, nil
}

func countStale(checks []packageCheck) int {
	n := 0
	for _, c := range checks {
		if c.needsRelease {
			n++
		}
	}
	return n
}

func printCheckTable(cmd *cobra.Command, checks []packageCheck) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PACKAGE\tVERSION\tLOCAL\tREMOTE\tSTATUS")

	for _, c := range checks {
		status := "ok"
		if c.needsRelease {
			status = "needs release"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			c.pkg.Name, c.version,
			shortSHA(c.localSHA), shortSHA(c.remoteSHA),
			status)
	}
	w.Flush() //nolint:errcheck
}

// resolveAppVersion reads APP_VERSION from .versions and resolves the next version.
func resolveAppVersion(repoRoot, explicit string) (string, string, error) {
	currentStr, err := version.ReadDotVersion(repoRoot, "APP_VERSION")
	if err != nil {
		return "", "", err
	}

	current, err := version.Parse(currentStr)
	if err != nil {
		return "", "", fmt.Errorf("APP_VERSION in .versions: %w", err)
	}

	if explicit == "" {
		next := current.BumpPatch()
		return currentStr, next.String(), nil
	}

	next, err := version.Parse(explicit)
	if err != nil {
		return "", "", err
	}

	if !next.GreaterThan(current) {
		return "", "", fmt.Errorf("version %s must be greater than current %s", next, current)
	}

	return currentStr, next.String(), nil
}

func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
