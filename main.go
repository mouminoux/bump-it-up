package main

import (
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
		do(&githubInfo, &repositoryInfo, mvnGroupId)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func do(githubInfo *github.GithubInfo, mavenRepositoryInfo *maven.RepositoryInfo, mavenGroupIdFilter *string) {
	repo, err := github.GetRepo(githubInfo)
	if err != nil {
		log.Fatal(err)
	}

	defer repo.DeleteRepo()

	pomPath := repo.GetTmpRepoPath() + "/pom.xml"
	dependencies := maven.ReadPom(pomPath)

	propertyNameAlreadyBumped := make(map[string]bool)
	for _, dependency := range dependencies {
		if alreadyBumped := propertyNameAlreadyBumped[dependency.PropertyName]; strings.Contains(dependency.GroupId, *mavenGroupIdFilter) && !alreadyBumped {
			lastVersion := maven.GetLastVersion(dependency, mavenRepositoryInfo)
			if lastVersion != dependency.Version {
				log.Printf("Bump %s:%s from %s to %s (%s)\n", dependency.GroupId, dependency.ArtifactId, dependency.Version, lastVersion, dependency.PropertyName)

				maven.ChangeVersion(pomPath, dependency, lastVersion)

				branchName := "bump-it-up/" + dependency.PropertyName + "/" + lastVersion

				if err := repo.PushAndCreatePR(branchName, "[BumpItUp] Bump "+dependency.PropertyName); err != nil {
					if strings.HasPrefix(err.Error(), "non-fast-forward update") {
						log.Printf("Impossible to bump version, maybe the branch %s already exist\n", branchName)
					}
					log.Printf("%v\n", err)
				}

				propertyNameAlreadyBumped[dependency.PropertyName] = true
			}
		}
	}
}
