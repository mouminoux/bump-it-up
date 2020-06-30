Bump It Up updates your maven dependencies and creates Github pull request.

Usage:
```
Usage: bump-it-up [OPTIONS]

Bump version
                                    
Options:                            
      --dry-run                     dry run: do not create pull request (env $DRY_RUN)
      --one-branch-per-dependency   one-branch-per-dependency: create one commit and one branch per dependency update (env $ONE_BRANCH_PER_DEPENDENCY) (default true)
      --access-token                github access token (env $ACCESS_TOKEN)
      --repository-owner            github repository owner (env $REPOSITORY_OWNER)
      --repository                  github repository name (env $REPOSITORY)
      --maven-repository-url        maven repository url (env $MAVEN_REPOSITORY_URL)
      --maven-repository-username   maven repository username (env $MAVEN_REPOSITORY_USER)
      --maven-repository-password   maven repository password (env $MAVEN_REPOSITORY_PASSWORD)
      --maven-group-id              maven group id filter (env $MAVEN_GROUP_ID)
```
