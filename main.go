package main

import (
	"fmt"
	"github.com/jawher/mow.cli"
	"github.com/mouminoux/bump-it-up/github"
	"github.com/mouminoux/bump-it-up/maven"
	"log"
	"os"
	"strings"
)

func main() {
	app := cli.App("bump-it-up", "Bump version")
	app.Spec = ""

	dryRun := app.Bool(cli.BoolOpt{
		Name:   "dry-run",
		Desc:   "dry run: do not create pull request",
		EnvVar: "DRY_RUN",
		Value:  false,
	})

	oneBranchPerDependency := app.Bool(cli.BoolOpt{
		Name:   "one-branch-per-dependency",
		Desc:   "one-branch-per-dependency: create one commit and one branch per dependency update",
		EnvVar: "ONE_BRANCH_PER_DEPENDENCY",
		Value:  true,
	})

	githubAccessToken := app.String(cli.StringOpt{
		Name:   "access-token",
		Desc:   "github access token",
		EnvVar: "ACCESS_TOKEN",
	})
	repoOwner := app.String(cli.StringOpt{
		Name:   "repository-owner",
		Desc:   "github repository owner",
		EnvVar: "REPOSITORY_OWNER",
	})
	repository := app.String(cli.StringOpt{
		Name:   "repository",
		Desc:   "github repository name",
		EnvVar: "REPOSITORY",
	})

	mvnUrl := app.String(cli.StringOpt{
		Name:   "maven-repository-url",
		Desc:   "maven repository url",
		EnvVar: "MAVEN_REPOSITORY_URL",
	})
	mvnUser := app.String(cli.StringOpt{
		Name:   "maven-repository-username",
		Desc:   "maven repository username",
		EnvVar: "MAVEN_REPOSITORY_USER",
	})
	mvnPasswd := app.String(cli.StringOpt{
		Name:   "maven-repository-password",
		Desc:   "maven repository password",
		EnvVar: "MAVEN_REPOSITORY_PASSWORD",
	})

	mvnGroupId := app.String(cli.StringOpt{
		Name:   "maven-group-id",
		Desc:   "maven group id filter",
		EnvVar: "MAVEN_GROUP_ID",
	})

	githubInfo := github.GithubInfo{
		AccessToken: *githubAccessToken,
		Owner:       *repoOwner,
		Repository:  *repository,
	}

	repositoryInfo := maven.RepositoryInfo{
		Url:      *mvnUrl,
		Username: *mvnUser,
		Password: *mvnPasswd,
	}

	app.Action = func() {
		do(&githubInfo, &repositoryInfo, mvnGroupId, dryRun, oneBranchPerDependency)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func do(githubInfo *github.GithubInfo, mavenRepositoryInfo *maven.RepositoryInfo, mavenGroupIdFilter *string, dryRun *bool, oneBranchPerDependency *bool) {
	if *dryRun {
		log.Printf("Dry run enabled")
	}

	repo, err := github.GetRepo(githubInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.DeleteRepo()

	pomPath := repo.GetTmpRepoPath() + "/pom.xml"
	dependencies := maven.ReadPom(pomPath)

	propertyNameAlreadyBumped := make(map[string]bool)

	prDescription := ""

	for _, dependency := range dependencies {

		if alreadyBumped := propertyNameAlreadyBumped[dependency.PropertyName]; !strings.Contains(dependency.GroupId, *mavenGroupIdFilter) || alreadyBumped {
			continue
		}

		propertyNameAlreadyBumped[dependency.PropertyName] = true

		lastVersion := maven.GetLastVersion(dependency, mavenRepositoryInfo)
		if lastVersion == dependency.Version {
			continue
		}

		log.Println(strings.Repeat("-", 60))

		var dependencyWithSamePropertyName []maven.Dependency
		for _, d := range dependencies {
			if d.PropertyName == dependency.PropertyName {
				dependencyWithSamePropertyName = append(dependencyWithSamePropertyName, d)
			}
		}

		prTitle := fmt.Sprintf("Bump %s from %s to %s", dependency.PropertyName, dependency.Version, lastVersion)
		log.Println(prTitle)

		for _, d := range dependencyWithSamePropertyName {
			depDesc := fmt.Sprintf("- Update %s:%s from %s to %s\n", d.GroupId, d.ArtifactId, d.Version, lastVersion)
			log.Print(depDesc)
			prDescription += depDesc
		}

		if *dryRun {
			continue
		}

		if err := maven.ChangeVersion(pomPath, dependency, lastVersion); err != nil {
			log.Fatalf("%v\n", err)
		}

		if *oneBranchPerDependency {
			branchName := "bump-it-up/" + dependency.PropertyName + "/" + lastVersion

			if err := repo.PushAndCreatePR(branchName, prTitle, prDescription); err != nil {
				log.Printf("%v\n", err)
			}
		}
	}

	if !*oneBranchPerDependency {
		if *dryRun {
			if err := repo.PushAndCreatePR("bump-it-up/bump-them-all", "Bump dependency with group-id: "+*mavenGroupIdFilter, prDescription); err != nil {
				log.Printf("%v\n", err)
			}
		} else {
			log.Printf("%v\n", prDescription)
		}
	}
}
