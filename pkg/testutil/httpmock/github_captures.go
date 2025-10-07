package httpmock

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"encoding/json"

	"github.com/google/go-github/v60/github"

	//nolint:staticcheck
	. "github.com/onsi/gomega"
)

type ActionVariableRequest struct {
	Org      string
	RepoName string
	Var      github.ActionsVariable
}

func NewCreateActionVariablesCapture() *HttpCaptureHandler[ActionVariableRequest] {
	return NewCaptureHandlerWithMappingFns(
		func(r *http.Request) ActionVariableRequest {
			body, err := io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred())
			var v github.ActionsVariable
			Expect(json.Unmarshal(body, &v)).To(Succeed())

			urlPathParts := strings.Split(r.URL.Path, "/")
			return ActionVariableRequest{
				Org:      urlPathParts[2],
				RepoName: urlPathParts[3],
				Var:      v,
			}
		},
		func(r *ActionVariableRequest) any {
			// GitHub API doesn't return anything
			return nil
		})
}

type CreateUpdateEnvRequest struct {
	Org      string
	RepoName string
	EnvName  string
	Env      github.CreateUpdateEnvironment
}

func NewCreateUpdateEnvCapture() *HttpCaptureHandler[CreateUpdateEnvRequest] {
	return NewCaptureHandlerWithMappingFns(
		func(r *http.Request) CreateUpdateEnvRequest {
			body, err := io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred())
			var e github.CreateUpdateEnvironment
			Expect(json.Unmarshal(body, &e)).To(Succeed())

			urlPathParts := strings.Split(r.URL.Path, "/")
			return CreateUpdateEnvRequest{
				Org:      urlPathParts[2],
				RepoName: urlPathParts[3],
				EnvName:  urlPathParts[5],
				Env:      e,
			}
		},
		func(r *CreateUpdateEnvRequest) any {
			// GitHub API doesn't return anything
			return github.Environment{
				Name:            &r.EnvName,
				EnvironmentName: &r.EnvName,
				Owner:           &r.Org,
				Repo:            &r.RepoName,
			}
		})
}

type ActionEnvVariableRequest struct {
	RepoID  int64
	EnvName string
	Var     github.ActionsVariable
}

func NewCreateActionEnvVariablesCapture() *HttpCaptureHandler[ActionEnvVariableRequest] {
	return NewCaptureHandlerWithMappingFns(
		func(r *http.Request) ActionEnvVariableRequest {
			body, err := io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred())
			var v github.ActionsVariable
			Expect(json.Unmarshal(body, &v)).To(Succeed())

			urlPathParts := strings.Split(r.URL.Path, "/")
			id, err := strconv.Atoi(urlPathParts[2])
			Expect(err).NotTo(HaveOccurred())
			return ActionEnvVariableRequest{
				RepoID:  int64(id),
				EnvName: urlPathParts[4],
				Var:     v,
			}
		},
		func(r *ActionEnvVariableRequest) any {
			// GitHub API doesn't return anything
			return nil
		})
}
