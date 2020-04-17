package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hauke96/sigolo"
	"io/ioutil"
	"net/http"
	"strings"

	"../auth"
	"../project"
	"../task"
	"../util"
)

func Init_V1(router *mux.Router) (*mux.Router, string) {
	routerV1 := router.PathPrefix("/v1").Subrouter()

	routerV1.HandleFunc("/projects", authenticatedHandler(getProjects)).Methods(http.MethodGet)
	routerV1.HandleFunc("/projects", authenticatedHandler(addProject)).Methods(http.MethodPost)
	routerV1.HandleFunc("/projects/users", authenticatedHandler(addUserToProject)).Methods(http.MethodPost)
	routerV1.HandleFunc("/tasks", authenticatedHandler(getTasks)).Methods(http.MethodGet)
	routerV1.HandleFunc("/tasks", authenticatedHandler(addTask)).Methods(http.MethodPost)
	routerV1.HandleFunc("/task/assignedUser", authenticatedHandler(assignUser)).Methods(http.MethodPost)
	routerV1.HandleFunc("/task/assignedUser", authenticatedHandler(unassignUser)).Methods(http.MethodDelete)
	routerV1.HandleFunc("/task/processPoints", authenticatedHandler(setProcessPoints)).Methods(http.MethodPost)

	return routerV1, "v1"
}

func getProjects(w http.ResponseWriter, r *http.Request, token *auth.Token) {
	projects, err := project.GetProjects(token.User)
	if err != nil {
		util.ResponseInternalError(w, err.Error())
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(projects)
}

func addProject(w http.ResponseWriter, r *http.Request, token *auth.Token) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errMsg := fmt.Sprintf("Error reading request body: %s", err.Error())
		sigolo.Error(errMsg)
		util.ResponseBadRequest(w, errMsg)
		return
	}

	var draftProject project.Project
	json.Unmarshal(bodyBytes, &draftProject)

	updatedProject, err := project.AddProject(&draftProject, token.User)
	if err != nil {
		util.ResponseInternalError(w, err.Error())
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(updatedProject)
}

func addUserToProject(w http.ResponseWriter, r *http.Request, token *auth.Token) {
	userName, err := util.GetParam("user", r)
	if err != nil {
		util.ResponseBadRequest(w, err.Error())
		return
	}

	projectId, err := util.GetParam("project", r)
	if err != nil {
		util.ResponseBadRequest(w, err.Error())
		return
	}

	updatedProject, err := project.AddUser(userName, projectId, token.User)
	if err != nil {
		util.ResponseInternalError(w, err.Error())
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(updatedProject)
}

func getTasks(w http.ResponseWriter, r *http.Request, token *auth.Token) {
	// Read task IDs from URL query parameter "task_ids" and split by ","
	taskIdsString, err := util.GetParam("task_ids", r)
	if err != nil {
		util.ResponseBadRequest(w, err.Error())
		return
	}

	taskIds := strings.Split(taskIdsString, ",")

	userOwnsTasks, err := project.VerifyOwnership(token.User, taskIds)
	if err != nil {
		util.ResponseInternalError(w, err.Error())
		return
	}
	if !userOwnsTasks {
		sigolo.Error("At least one task belongs to a project where the user '%s' is not part of", token.User)
		util.Response(w, "Not all tasks belong to user", http.StatusForbidden)
		return
	}

	tasks, err := task.GetTasks(taskIds)
	if err != nil {
		util.ResponseInternalError(w, err.Error())
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(tasks)
}

func addTask(w http.ResponseWriter, r *http.Request, token *auth.Token) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sigolo.Error("Error reading request body: %s", err.Error())
		util.ResponseBadRequest(w, err.Error())
		return
	}

	var tasks []*task.Task
	json.Unmarshal(bodyBytes, &tasks)

	updatedTasks, err := task.AddTasks(tasks)
	if err != nil {
		util.ResponseInternalError(w, err.Error())
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(updatedTasks)
}

func assignUser(w http.ResponseWriter, r *http.Request, token *auth.Token) {
	taskId, err := util.GetParam("id", r)
	if err != nil {
		sigolo.Error(err.Error())
		util.ResponseBadRequest(w, err.Error())
		return
	}

	user := token.User

	task, err := task.AssignUser(taskId, user)
	if err != nil {
		sigolo.Error(err.Error())
		util.ResponseInternalError(w, err.Error())
		return
	}

	sigolo.Info("Successfully assigned user '%s' to task '%s'", user, taskId)

	encoder := json.NewEncoder(w)
	encoder.Encode(*task)
}

func unassignUser(w http.ResponseWriter, r *http.Request, token *auth.Token) {
	taskId, err := util.GetParam("id", r)
	if err != nil {
		sigolo.Error(err.Error())
		util.ResponseBadRequest(w, err.Error())
		return
	}

	user := token.User

	task, err := task.UnassignUser(taskId, user)
	if err != nil {
		sigolo.Error(err.Error())
		util.ResponseInternalError(w, err.Error())
		return
	}

	sigolo.Info("Successfully unassigned user '%s' from task '%s'", user, taskId)

	encoder := json.NewEncoder(w)
	encoder.Encode(*task)
}

func setProcessPoints(w http.ResponseWriter, r *http.Request, token *auth.Token) {
	taskId, err := util.GetParam("id", r)
	if err != nil {
		sigolo.Error(err.Error())
		util.ResponseBadRequest(w, err.Error())
		return
	}

	processPoints, err := util.GetIntParam("process_points", w, r)
	if err != nil {
		sigolo.Error(err.Error())
		util.ResponseBadRequest(w, err.Error())
		return
	}

	task, err := task.SetProcessPoints(taskId, processPoints)
	if err != nil {
		sigolo.Error(err.Error())
		util.ResponseInternalError(w, err.Error())
		return
	}

	sigolo.Info("Successfully set process points on task '%s' to %d", taskId, processPoints)

	encoder := json.NewEncoder(w)
	encoder.Encode(*task)
}
