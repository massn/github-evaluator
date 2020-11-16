package stats

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v32/github"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"strings"
)

type Repo struct {
	Name     string `yaml:"name"`
	Location string `yaml:"location"`
}

func Stats(reposYamlPath string) {
	data, err := ioutil.ReadFile(reposYamlPath)
	if err != nil {
		panic(err)
	}
	var repos []Repo
	err = yaml.Unmarshal(data, &repos)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d repos\n", len(repos))
	ctx := context.Background()
	client := github.NewClient(nil)
	for _, repo := range repos {
		owner, project, err := getOwnerAndProject(repo.Location)
		if err != nil {
			fmt.Println(err)
			continue
		}
		csList, _, err := client.Repositories.ListContributorsStats(ctx, owner, project)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("location : %s, %d contibutors", repo.Location, len(csList))
	}
}

func getOwnerAndProject(location string) (string, string, error) {
	slice := strings.Split(location, "/")
	if len(slice) != 2 {
		return "", "", errors.New("Failed to get owner and project from " + location)
	}
	return slice[0], slice[1], nil
}
