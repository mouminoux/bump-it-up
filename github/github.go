package github

import (
	"context"
	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"log"
	"time"

	//	"github.com/google/go-github/v29/github"
	//	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-git.v4"
	"os"
)

type Github struct {
	repo        *git.Repository
	auth        *http.BasicAuth
	githubInfo  GithubInfo
	tmpRepoPath string
}

type GithubInfo struct {
	AccessToken string
	Owner       string
	Repository  string
}

func GetRepo(githubInfo *GithubInfo) (*Github, error) {
	tmpRepoPath := "/tmp/bump-it-up"
	_ = os.RemoveAll(tmpRepoPath)

	auth := &http.BasicAuth{
		Username: "none", // anything except an empty string
		Password: githubInfo.AccessToken,
	}

	repo, err := git.PlainClone(tmpRepoPath, false, &git.CloneOptions{
		URL:      "https://github.com/" + githubInfo.Owner + "/" + githubInfo.Repository,
		Progress: os.Stdout,
		Depth:    0,
		Auth:     auth,
	})
	if err != nil {
		return nil, err
	}

	g := &Github{
		repo:           repo,
		auth:           auth,
		tmpRepoPath:    tmpRepoPath,
		githubInfo: *githubInfo,
	}
	return g, nil
}

func (g *Github) GetTmpRepoPath() string {
	return g.tmpRepoPath
}

func (g *Github) DeleteRepo() {
	defer func() {
		_ = os.RemoveAll(g.tmpRepoPath)
	}()
}

func (g *Github) PushAndCreatePR(branchName string, title string) error {

	if _, err := g.checkoutBranch("master"); err != nil {
		return err
	}

	// create branch
	if err := g.createBranch(branchName); err != nil {
		return err
	}

	//create commit
	gitWorkTree, err := g.checkoutBranch(branchName)
	if err != nil {
		return err
	}

	_, err = gitWorkTree.Add(".")
	if err != nil {
		return err
	}

	_, err = gitWorkTree.Commit(title, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Bump It Up",
			Email: "bump-it-up",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	// push
	pushOpt := &git.PushOptions{
		RefSpecs: nil,
		Auth:     g.auth,
		Progress: nil,
		Prune:    false,
	}
	if err := g.repo.Push(pushOpt); err != nil {
		return err
	}

	// create PR
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.auth.Password},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	master := "master"
	pullRequest, _, err := client.PullRequests.Create(ctx, g.githubInfo.Owner, g.githubInfo.Repository, &github.NewPullRequest{
		Title:               &title,
		Head:                &branchName,
		Base:                &master,
		Body:                nil,
		Issue:               nil,
		MaintainerCanModify: nil,
		Draft:               nil,
	})
	if err != nil {
		return err
	}

	log.Printf("Pull request created: %s\n", *pullRequest.HTMLURL)

	if _, err = g.checkoutBranch("master"); err != nil {
		return err
	}

	return nil
}

func (g *Github) checkoutBranch(branchName string) (*git.Worktree, error) {
	gitWorkTree, err := g.repo.Worktree()
	if err != nil {
		return nil, err
	}
	err = gitWorkTree.Checkout(&git.CheckoutOptions{
		Hash:   plumbing.Hash{},
		Branch: plumbing.NewBranchReferenceName(branchName),
		Create: false,
		Force:  false,
		Keep:   true,
	})
	if err != nil {
		return nil, err
	}
	return gitWorkTree, nil
}

func (g *Github) createBranch(branchName string) error {
	headRef, err := g.repo.Head()
	if err != nil {
		return err
	}
	ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), headRef.Hash())
	err = g.repo.Storer.SetReference(ref)
	if err != nil {
		return err
	}
	return nil
}
