package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	gogithub "github.com/google/go-github/v68/github"
)

// Client defines the GitHub operations needed by the deploy pipeline.
type Client interface {
	RepoExists(ctx context.Context, repo string) (bool, error)
	EnsureRepoExists(ctx context.Context, repo string) error
	CreateRelease(ctx context.Context, repo, tag, notes string) error
	GetRef(ctx context.Context, repo, ref string) (sha string, err error)
}

// ghClient is the production implementation backed by go-github.
type ghClient struct {
	client *gogithub.Client
	token  string
	owner  string
}

// NewClient creates a GitHub client using the provided owner and token.
func NewClient(owner, token string) Client {
	c := gogithub.NewClient(nil).WithAuthToken(token)
	return &ghClient{
		client: c,
		token:  token,
		owner:  owner,
	}
}

// NewClientFromEnv creates a GitHub client reading the token from GITHUB_ACCESS_TOKEN.
func NewClientFromEnv(owner string) (Client, error) {
	token := os.Getenv("GITHUB_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_ACCESS_TOKEN is required")
	}
	return NewClient(owner, token), nil
}

func (g *ghClient) RepoExists(ctx context.Context, repo string) (bool, error) {
	_, resp, err := g.client.Repositories.Get(ctx, g.owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("check repo %s/%s: %w", g.owner, repo, err)
	}
	return true, nil
}

func (g *ghClient) EnsureRepoExists(ctx context.Context, repo string) error {
	exists, err := g.RepoExists(ctx, repo)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("mirror repo %s/%s does not exist on GitHub\n  create it at: https://github.com/organizations/%s/repositories/new?name=%s",
			g.owner, repo, g.owner, repo)
	}
	return nil
}

func (g *ghClient) CreateRelease(ctx context.Context, repo, tag, notes string) error {
	_, _, err := g.client.Repositories.CreateRelease(ctx, g.owner, repo, &gogithub.RepositoryRelease{
		TagName: gogithub.Ptr(tag),
		Name:    gogithub.Ptr(tag),
		Body:    gogithub.Ptr(notes),
	})
	if err != nil {
		return fmt.Errorf("create release %s on %s/%s: %w", tag, g.owner, repo, err)
	}
	return nil
}

func (g *ghClient) GetRef(ctx context.Context, repo, ref string) (string, error) {
	r, resp, err := g.client.Git.GetRef(ctx, g.owner, repo, ref)
	if err != nil {
		if resp != nil && (resp.StatusCode == 404 || resp.StatusCode == 409) {
			return "", nil
		}
		return "", fmt.Errorf("get ref %s on %s/%s: %w", ref, g.owner, repo, err)
	}
	return r.GetObject().GetSHA(), nil
}

// PushGit pushes a local SHA to the mirror repo via git push.
// refs are full ref names like "refs/heads/main" or "refs/tags/v0.1.0".
func PushGit(repoRoot, token, owner, repo, sha string, refs []string, dryRun bool) error {
	remoteURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo)

	refSpecs := make([]string, len(refs))
	for i, ref := range refs {
		refSpecs[i] = sha + ":" + ref
	}

	if dryRun {
		safeURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
		fmt.Fprintf(os.Stderr, "  [dry-run] would push %s to %s %s\n", sha[:12], safeURL, strings.Join(refs, " "))
		return nil
	}

	args := append([]string{"push", remoteURL}, refSpecs...)
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push to %s/%s: %w", owner, repo, err)
	}
	return nil
}
