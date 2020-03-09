package maven

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/mcuadros/go-version"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
)

type metadata struct {
	XMLName    xml.Name `xml:"metadata"`
	Text       string   `xml:",chardata"`
	GroupId    string   `xml:"groupId"`
	ArtifactId string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Versioning struct {
		Text     string `xml:",chardata"`
		Latest   string `xml:"latest"`
		Release  string `xml:"release"`
		Versions struct {
			Text    string   `xml:",chardata"`
			Version []string `xml:"version"`
		} `xml:"versions"`
		LastUpdated string `xml:"lastUpdated"`
	} `xml:"versioning"`
}

type project struct {
	XMLName        xml.Name `xml:"project"`
	Text           string   `xml:",chardata"`
	Xmlns          string   `xml:"xmlns,attr"`
	Xsi            string   `xml:"xsi,attr"`
	SchemaLocation string   `xml:"schemaLocation,attr"`
	Properties     struct {
		Property []property `xml:",any"`
	} `xml:"properties"`
	DependencyManagement struct {
		Text         string `xml:",chardata"`
		Dependencies struct {
			Text       string       `xml:",chardata"`
			Dependency []Dependency `xml:"dependency"`
		} `xml:"dependencies"`
	} `xml:"dependencyManagement"`
}

type property struct {
	XMLName xml.Name `xml:""`
	Version string   `xml:",chardata"`
}

type Dependency struct {
	GroupId      string `xml:"groupId"`
	ArtifactId   string `xml:"artifactId"`
	Version      string `xml:"version"`
	PropertyName string
}

type RepositoryInfo struct {
	Url      string
	Username string
	Password string
}

func ReadPom(filePath string) []Dependency {
	xmlFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
	}

	byteValue, _ := ioutil.ReadAll(xmlFile)

	var project project
	err = xml.Unmarshal(byteValue, &project)
	if err != nil {
		fmt.Println(err)
	}

	propertyMap := make(map[string]string)
	for _, property := range *(&project.Properties.Property) {
		//fmt.Printf("%s, %s\n", property.XMLName.Local, property.Version)
		propertyMap[property.XMLName.Local] = property.Version
	}

	var dependencies []Dependency

	alreadyAdded := make(map[string]struct{})

	for _, dependency := range project.DependencyManagement.Dependencies.Dependency {
		propertyName := strings.Replace(strings.Replace(dependency.Version, "${", "", 1), "}", "", 1)
		//fmt.Printf("%s, %s, %s\n", dependency.ArtifactId, dependency.GroupId, propertyMap[propertyName])

		// ignore project.version property
		if propertyName == "project.version" {
			continue
		}

		// no property detected
		if propertyName == dependency.Version {
			dependencies = append(dependencies, Dependency{
				GroupId:      dependency.GroupId,
				ArtifactId:   dependency.ArtifactId,
				Version:      dependency.Version,
				PropertyName: "",
			})
			continue
		}

		if _, ok := alreadyAdded[propertyName]; ok {
			//continue
		}

		dependencies = append(dependencies, Dependency{
			GroupId:      dependency.GroupId,
			ArtifactId:   dependency.ArtifactId,
			Version:      propertyMap[propertyName],
			PropertyName: propertyName,
		})

		alreadyAdded[propertyName] = struct{}{}
	}

	return dependencies
}

func GetLastVersion(dependency Dependency, repositoryInfo *RepositoryInfo) string {
	groupIdPath := strings.Replace(dependency.GroupId, ".", "/", -1)
	artifactIdPath := dependency.ArtifactId

	client := &http.Client{}
	req, err := http.NewRequest("GET", repositoryInfo.Url+"/"+groupIdPath+"/"+artifactIdPath+"/maven-metadata.xml", nil)
	if err != nil {
		fmt.Println(err)
	}
	req.SetBasicAuth(repositoryInfo.Username, repositoryInfo.Password)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	metadataXml, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	var metadata metadata
	err = xml.Unmarshal(metadataXml, &metadata)
	if err != nil {
		fmt.Println(err)
	}

	return findLastest(metadata.Versioning.Versions.Version, dependency.Version, struct{}{})
}

func ChangeVersion(filePath string, dependency Dependency, newVersion string) error {
	oldVersionString := "<" + dependency.PropertyName + ">" + dependency.Version + "</" + dependency.PropertyName + ">"
	newVersionString := "<" + dependency.PropertyName + ">" + newVersion + "</" + dependency.PropertyName + ">"

	input, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	output := bytes.Replace(input, []byte(oldVersionString), []byte(newVersionString), 1)

	if err = ioutil.WriteFile(filePath, output, 0666); err != nil {
		return err
	}

	return nil
}

func findLastest(metadataVersions []string, currentVersion string, rules struct{}) string {
	metadataVersionsWithoutSnapshots := make(map[string]string)

	for _, metadataVersion := range metadataVersions {
		if !strings.Contains(metadataVersion, "-SNAPSHOT") {
			normalizedVersion := version.Normalize(metadataVersion)
			metadataVersionsWithoutSnapshots[normalizedVersion] = metadataVersion
		}
	}

	if len(metadataVersionsWithoutSnapshots) == 0 {
		return ""
	}

	var keys []string
	for k := range metadataVersionsWithoutSnapshots {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return version.Compare(keys[i], keys[j], ">")
	})

	return metadataVersionsWithoutSnapshots[keys[0]]
}
