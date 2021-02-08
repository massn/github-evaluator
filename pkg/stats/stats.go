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
	Etc          string `yaml:"etc"`
	StarsHistory string
	Error        error
}

type StatsClient struct {
	client  *github.Client
	ctx     context.Context
	repo    Repo
	owner   string
	project string
}

type mode int

const (
	History mode = iota
	Contributors
	Info
	Issues
)

func Stats(reposYamlPath string, m mode) error {
	var gettingFunc func(StatsClient, chan Repo)
	switch m {
	case History:
		gettingFunc = getStarsHistory
	case Contributors:
		gettingFunc = getContributors
	case Info:
		gettingFunc = getInformation
	case Issues:
		gettingFunc = getIssues
	default:
		return fmt.Errorf("Unknown mode:%d", m)
	}

	repos, err := readRepos(reposYamlPath)
	if err != nil {
		return err
	}

	fmt.Printf("%d repos\n", len(repos))
	resultRepos := []Repo{}

	repoChan := make(chan Repo, len(repos))
	for _, repo := range repos {
		sc, err := newStatsClient(repo)
		if err != nil {
			return err
		}
		go gettingFunc(sc, repoChan)
	}
	for range repos {
		resultRepo := <-repoChan
		resultRepos = append(resultRepos, resultRepo)
	}

	var s []Repo
	switch m {
	case Contributors:
		s = sortInContributorsDecending(resultRepos)
	case Issues:
		s = sortInIssuesDecending(resultRepos)
	default:
		s = sortInStarsDecending(resultRepos)
	}
	printTable(s)
	return nil
}

func readRepos(reposYamlPath string) ([]Repo, error) {
	data, err := ioutil.ReadFile(reposYamlPath)
	if err != nil {
		return []Repo{}, err
	}
	var repos []Repo
	err = yaml.Unmarshal(data, &repos)
	if err != nil {
		return []Repo{}, err
	}
	return repos, nil
}

func sortInIssuesDecending(repos []Repo) []Repo {
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Issues > repos[j].Issues
	})
	return repos
}

func sortInStarsDecending(repos []Repo) []Repo {
	sort.Slice(repos, func(i, j int) bool {
		if repos[i].Information == nil || repos[j].Information == nil {
			return true
		}
		return *repos[i].Information.StargazersCount > *repos[j].Information.StargazersCount
	})
	return repos
}

func sortInContributorsDecending(repos []Repo) []Repo {
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Contributors > repos[j].Contributors
	})
	return repos
}

func printTable(repos []Repo) {
	tableData := [][]string{}
	for _, repo := range repos {
		var stargazersCount string
		if repo.Information == nil {
			stargazersCount = "unknown"
		} else {
			stargazersCount = strconv.Itoa(*repo.Information.StargazersCount)
		}

		entry := []string{
			repo.Name,
			repo.Location,
			strconv.Itoa(repo.Contributors),
			strconv.Itoa(repo.Issues),
			stargazersCount,
			repo.Etc,
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
		"Etc",
	})
	table.SetFooter([]string{"", "", "", "", "", time.Now().Local().Format("2006-01-02 15:04:05")})
	table.AppendBulk(tableData)
	table.Render()
}

func newStatsClient(repo Repo) (StatsClient, error) {
	ctx := context.Background()
	client := newClient(ctx)
	owner, project, err := getOwnerAndProject(repo.Location)
	if err != nil {
		fmt.Printf("Failed to get the owner and project of %v. Reason: %v ", repo.Name, err)
		return StatsClient{}, err
	}
	return StatsClient{
		client:  client,
		ctx:     ctx,
		owner:   owner,
		project: project,
		repo:    repo,
	}, nil
}

func newClient(ctx context.Context) *github.Client {
	data, err := ioutil.ReadFile("token")
	if err != nil {
		fmt.Println("Failed to get token")
		panic(err)
	}
	token := strings.Trim(string(data), "\n")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func getStarsHistory(sc StatsClient, repoChan chan Repo) {
	// TODO implement
	//github.ActivityService.ListStargazsers(sc.ctx, sc.owner, sc.repo)
	repoChan <- sc.repo
}

func getContributors(sc StatsClient, repoChan chan Repo) {
	perPage := 100
	l := github.ListOptions{PerPage: perPage}
	conOpts := github.ListContributorsOptions{ListOptions: l}

	csList, resp, err := sc.client.Repositories.ListContributors(sc.ctx, sc.owner, sc.project, &conOpts)
	if err != nil {
		repoChan <- sc.repo
		return
	}
	fp := resp.FirstPage
	lp := resp.LastPage
	if fp == lp {
		sc.repo.Contributors = len(csList)
		repoChan <- sc.repo
		return
	}

	ll := github.ListOptions{PerPage: perPage, Page: lp}
	conOpts = github.ListContributorsOptions{ListOptions: ll}
	lastCsList, _, err := sc.client.Repositories.ListContributors(sc.ctx, sc.owner, sc.project, &conOpts)
	if err != nil {
		sc.repo.Error = err
		repoChan <- sc.repo
		return
	}
	sc.repo.Contributors = perPage*(lp-fp) + len(lastCsList)
	repoChan <- sc.repo
}

func getIssues(sc StatsClient, repoChan chan Repo) {
	perPage := 30
	l := github.ListOptions{PerPage: perPage}
	repoOpts := github.IssueListByRepoOptions{State: "all", ListOptions: l}

	isList, resp, err := sc.client.Issues.ListByRepo(sc.ctx, sc.owner, sc.project, &repoOpts)
	if err != nil {
		sc.repo.Error = err
		repoChan <- sc.repo
		return
	}
	fp := resp.FirstPage
	lp := resp.LastPage
	if fp == lp {
		sc.repo.Issues = len(isList)
		repoChan <- sc.repo
		return
	}

	ll := github.ListOptions{PerPage: perPage, Page: lp}
	repoOpts = github.IssueListByRepoOptions{State: "all", ListOptions: ll}
	lastIsList, _, err := sc.client.Issues.ListByRepo(sc.ctx, sc.owner, sc.project, &repoOpts)
	if err != nil {
		sc.repo.Error = err
		repoChan <- sc.repo
		return
	}
	sc.repo.Issues = perPage*(lp-fp) + len(lastIsList)
	repoChan <- sc.repo
}

func getInformation(sc StatsClient, repoChan chan Repo) {
	info, _, err := sc.client.Repositories.Get(sc.ctx, sc.owner, sc.project)
	if err != nil {
		sc.repo.Error = err
		repoChan <- sc.repo
		return
	}
	sc.repo.Information = info
	repoChan <- sc.repo
	return
}

func getOwnerAndProject(location string) (string, string, error) {
	slice := strings.Split(location, "/")
	if len(slice) != 2 {
		return "", "", errors.New("Failed to get owner and project from " + location)
	}
	return slice[0], slice[1], nil
}
