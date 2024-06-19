package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func NewGHRepoVarStorage() (*GithubRepository, error) {
	v, ok := os.LookupEnv(EnvGithubPatVarName)
	if !ok || len(v) == 0 {
		return nil, fmt.Errorf("the environment variable for github pat is missing")
	}
	return &GithubRepository{githubToken: v, fullRepoName: "cncf-tags/green-reviews-tooling"}, nil
}

func (obj *GithubRepository) generateVariableName(projName string) string {
	return strings.ToLower(projName + "_version")
}

func (obj *GithubRepository) genUrlAndHeaders(
	endpointType githubApiEndpointType,
	variableName string, workflowFileName string,
) (*string, map[string]string, error) {
	url := ""

	switch endpointType {
	case ghRepoVariableEndpoint:
		url = fmt.Sprintf(
			string(endpointType),
			obj.fullRepoName,
			variableName,
		)
	case ghWorkflowDispatchEndpoint:
		url = fmt.Sprintf(
			string(endpointType),
			obj.fullRepoName,
			workflowFileName,
		)
	}

	return &url,
		map[string]string{
			"Accept":               "application/vnd.github+json",
			"Authorization":        "Bearer " + obj.githubToken,
			"X-GitHub-Api-Version": "2022-11-28",
		}, nil
}

func (obj *GithubRepository) UpdateRepoVariable(projName, newVersion string) error {
	variableName := obj.generateVariableName(projName)

	url, header, err := obj.genUrlAndHeaders(ghRepoVariableEndpoint, variableName, "")
	if err != nil {
		return err
	}

	newVariableData := struct {
		Name string `json:"name"`
		Val  string `json:"value"`
	}{
		Name: strings.ToUpper(variableName),
		Val:  newVersion,
	}

	var _newVariableData bytes.Buffer

	if err := json.NewEncoder(&_newVariableData).Encode(newVariableData); err != nil {
		return fmt.Errorf("failed to serialize the body: %v", err)
	}

	resp, err := handleHTTPCall(
		http.MethodPatch,
		*url,
		time.Minute,
		&_newVariableData,
		header,
	)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("status code was not 204, got: %v", resp.StatusCode)
	}

	return nil
}

func (obj *GithubRepository) ReadRepoVariable(projName string) (*string, error) {
	variableName := obj.generateVariableName(projName)
	url, header, err := obj.genUrlAndHeaders(ghRepoVariableEndpoint, variableName, "")
	if err != nil {
		return nil, err
	}

	resp, err := handleHTTPCall(
		http.MethodGet,
		*url,
		time.Minute,
		nil,
		header,
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code was not 200, got: %v", resp.StatusCode)
	}

	var variableData struct {
		VariableValue string `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&variableData); err != nil {
		return nil, fmt.Errorf("failed deserialize response body: %v", err)
	}

	return &variableData.VariableValue, nil
}

func fetchLatestRelease(org, proj string) (*string, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/releases/latest", org, proj)

	resp, err := handleHTTPCall(
		http.MethodGet,
		url,
		time.Minute, nil, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code was not 200, got: %v", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed deserialize response body: %v", err)
	}

	slog.Info("Latest Release", "Proj", proj, "Org", org, "Ver", release.TagName)

	return &release.TagName, nil
}
