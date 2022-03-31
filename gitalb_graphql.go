package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

var apiUrl = "https://gitlab.com/api/graphql"

var query = "query last_projects($repoCount: Int!) {projects(last:$repoCount) {nodes {name forksCount}}}"

type GitlabGraphQLClient struct {
	log  log.FieldLogger
	http *http.Client
}

func NewGitlabGraphQLClient(log log.FieldLogger, httpClient *http.Client) *GitlabGraphQLClient {
	return &GitlabGraphQLClient{
		log:  log,
		http: httpClient,
	}
}

func (client *GitlabGraphQLClient) FetchRepositoryData(repoCount int) ([]RepositoryData, error) {
	// Request body with GraphQL parameters.
	body := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query: query,
		Variables: map[string]interface{}{
			"repoCount": repoCount,
		},
	}
	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	client.log.Debugf("Requesting data for %d repos from Gitlab API\n", repoCount)

	// Send request.
	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(bodyData))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	client.log.Debugf("Data for %d repos received from Gitlab API\n", repoCount)

	// Read JSON resposne.
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	respJSON := struct {
		Data struct {
			Projects struct {
				Nodes []RepositoryData
			} `json:"projects"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(respData, &respJSON); err != nil {
		return nil, err
	}

	return respJSON.Data.Projects.Nodes, nil
}
