package stats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/oauth2"
	"io/ioutil"
	"os"
	"sort"

	"gopkg.in/yaml.v2"

	"strconv"
	"strings"
)

type Repo struct {
	Name         string `yaml:"name"`
	Location     string `yaml:"location"`
	Contributors int
	Issues       int
	Information  *github.Repository
}

type ClientParameter struct {
	client  *github.Client
	ctx     context.Context
	owner   string
	project string
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
	client := newClient(ctx)
	p := ClientParameter{ctx: ctx, client: client}
	resultRepos := []Repo{}

	repoChan := make(chan Repo, len(repos))

	for _, repo := range repos {
		go getStats(repo, p, repoChan)
	}
	for range repos {
		resultRepo := <-repoChan
		resultRepos = append(resultRepos, resultRepo)
	}
	sort.Slice(resultRepos, func(i, j int) bool {
		return *resultRepos[i].Information.StargazersCount > *resultRepos[j].Information.StargazersCount
	})

	tableData := [][]string{}
	for _, repo := range resultRepos {
		entry := []string{
			repo.Name,
			repo.Location,
			strconv.Itoa(repo.Contributors),
			strconv.Itoa(repo.Issues),
			strconv.Itoa(*repo.Information.StargazersCount),
		}
		tableData = append(tableData, entry)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"Name",
		"Location",
		"Contributors",
		"Issues",
		"Stars",
	})
	table.SetFooter([]string{"", "", "", "", time.Now().Local().Format("2006-01-02 15:04:05")})
	table.AppendBulk(tableData)
	table.Render()
}

func newClient(ctx context.Context) *github.Client {
	data, err := ioutil.ReadFile("token")
	if err != nil {
		fmt.Println("Failed to login")
		panic(err)
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(data)},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func getStats(repo Repo, p ClientParameter, repoChan chan Repo) {
	var resultRepo Repo
	resultRepo.Name = repo.Name
	resultRepo.Location = repo.Location
	owner, project, err := getOwnerAndProject(repo.Location)
	if err != nil {
		fmt.Printf("Failed to get the owner and project of %v. Reason: %v ", repo.Name, err)
		repoChan <- resultRepo
		return
	}
	p.owner = owner
	p.project = project
	info, err := getInformation(repo, p)
	if err != nil {
		fmt.Printf("Failed to get the information of %v. Reason: %v ", repo.Name, err)
		repoChan <- resultRepo
		return
	}
	resultRepo.Information = info
	cs, err := getContributors(repo, p)
	if err != nil {
		fmt.Printf("Failed to get the contributors of %v. Reason: %v ", repo.Name, err)
		repoChan <- resultRepo
		return
	}
	resultRepo.Contributors = cs

	issues, err := getIssues(repo, p)
	if err != nil {
		fmt.Printf("Failed to get the issues of %v. Reason: %v ", repo.Name, err)
		repoChan <- resultRepo
		return
	}
	resultRepo.Issues = issues
	repoChan <- resultRepo
}

func getContributors(repo Repo, p ClientParameter) (int, error) {
	perPage := 100
	l := github.ListOptions{PerPage: perPage}
	conOpts := github.ListContributorsOptions{ListOptions: l}

	csList, resp, err := p.client.Repositories.ListContributors(p.ctx, p.owner, p.project, &conOpts)
	if err != nil {
		return 0, err
	}
	fp := resp.FirstPage
	lp := resp.LastPage
	if fp == lp {
		return len(csList), nil
	}

	ll := github.ListOptions{PerPage: perPage, Page: lp}
	conOpts = github.ListContributorsOptions{ListOptions: ll}
	lastCsList, _, err := p.client.Repositories.ListContributors(p.ctx, p.owner, p.project, &conOpts)
	if err != nil {
		return 0, err
	}
	return perPage*(lp-fp) + len(lastCsList), nil
}

func getIssues(repo Repo, p ClientParameter) (int, error) {
	perPage := 30
	l := github.ListOptions{PerPage: perPage}
	repoOpts := github.IssueListByRepoOptions{State: "all", ListOptions: l}

	isList, resp, err := p.client.Issues.ListByRepo(p.ctx, p.owner, p.project, &repoOpts)
	if err != nil {
		return 0, err
	}
	fp := resp.FirstPage
	lp := resp.LastPage
	if fp == lp {
		return len(isList), nil
	}

	ll := github.ListOptions{PerPage: perPage, Page: lp}
	repoOpts = github.IssueListByRepoOptions{State: "all", ListOptions: ll}
	lastIsList, _, err := p.client.Issues.ListByRepo(p.ctx, p.owner, p.project, &repoOpts)
	if err != nil {
		return 0, err
	}
	return perPage*(lp-fp) + len(lastIsList), nil
}

func getInformation(repo Repo, p ClientParameter) (*github.Repository, error) {
	info, _, err := p.client.Repositories.Get(p.ctx, p.owner, p.project)
	if err != nil {
		return &github.Repository{}, err
	}
	return info, nil
}

func getOwnerAndProject(location string) (string, string, error) {
	slice := strings.Split(location, "/")
	if len(slice) != 2 {
		return "", "", errors.New("Failed to get owner and project from " + location)
	}
	return slice[0], slice[1], nil
}
