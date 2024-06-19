package main

const (
	LocProjMeta         string = "../projects/project.json"
	EnvGithubPatVarName string = "GH_TOKEN"
)

const (
	ghRepoVariableEndpoint     githubApiEndpointType = "https://api.github.com/repos/%s/actions/variables/%s"
	ghWorkflowDispatchEndpoint githubApiEndpointType = "https://api.github.com/repos/%s/actions/workflows/%s/dispatches"
)
