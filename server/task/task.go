package task

import (
	"database/sql"
	"fmt"
	"github.com/hauke96/sigolo"
	"github.com/hauke96/simple-task-manager/server/permission"
	"github.com/pkg/errors"
	"strings"
)

type Task struct {
	Id               string `json:"id"`
	ProcessPoints    int    `json:"processPoints"`
	MaxProcessPoints int    `json:"maxProcessPoints"`
	Geometry         string `json:"geometry"`
	AssignedUser     string `json:"assignedUser"`
}

type TaskService struct {
	store *storePg
	permissionService *permission.PermissionService
}

func Init(tx *sql.Tx, permissionService *permission.PermissionService) *TaskService {
	return &TaskService{
		store:             getStore(tx),
		permissionService: permissionService,
	}
}

// GetTasks checks the membership of the requesting user and gets the tasks requested by the IDs.
func (s *TaskService) GetTasks(taskIds []string, requestingUserId string) ([]*Task, error) {
	err := s.permissionService.VerifyMembershipTasks(taskIds, requestingUserId)
	if err != nil {
		return nil, err
	}

	return s.store.getTasks(taskIds)
}

// AddTasks sets the ID of the tasks and adds them to the storage.
func (s *TaskService) AddTasks(newTasks []*Task) ([]*Task, error) {
	for _, t := range newTasks {
		if t.ProcessPoints < 0 || t.MaxProcessPoints < 1 || t.MaxProcessPoints < t.ProcessPoints {
			return nil, errors.New(fmt.Sprintf("process points of task are out of range (%d / %d)", t.ProcessPoints, t.MaxProcessPoints))
		}
	}

	return s.store.addTasks(newTasks)
}

func (s *TaskService) AssignUser(taskId, userId string) (*Task, error) {
	task, err := s.store.getTask(taskId)
	if err != nil {
		return nil, err
	}

	// Task has already an assigned user
	if strings.TrimSpace(task.AssignedUser) != "" {
		return nil, errors.New(fmt.Sprintf("task %s has already an assigned userId, cannot overwrite", task.Id))
	}

	return s.store.assignUser(taskId, userId)
}

func (s *TaskService) UnassignUser(taskId, requestingUserId string) (*Task, error) {
	err := s.permissionService.VerifyAssignment(taskId, requestingUserId)
	if err != nil {
		return nil, err
	}

	return s.store.unassignUser(taskId)
}

// SetProcessPoints updates the process points on task "id". When "needsAssignedUser" is true on the project, this
// function also checks, whether the assigned user is equal to the requesting User.
func (s *TaskService) SetProcessPoints(taskId string, newPoints int, requestingUserId string) (*Task, error) {
	needsAssignment, err := s.permissionService.AssignmentInTaskNeeded(taskId)
	if err != nil {
		return nil, err
	}
	if needsAssignment {
		err := s.permissionService.VerifyAssignment(taskId, requestingUserId)
		if err != nil {
			return nil, err
		}
	} else { // when no assignment is needed, the requesting user at least needs to be a member
		err := s.permissionService.VerifyMembershipTask(taskId, requestingUserId)
		if err != nil {
			sigolo.Error("user not a member of the project, the task %s belongs to", taskId)
			return nil, err
		}
	}

	task, err := s.store.getTask(taskId)
	if err != nil {
		return nil, err
	}

	// New process points should be in the range "[0, MaxProcessPoints]" (so including 0 and MaxProcessPoints)
	if newPoints < 0 || task.MaxProcessPoints < newPoints {
		return nil, errors.New("process points out of range")
	}

	return s.store.setProcessPoints(taskId, newPoints)
}

// Delete will remove the given tasks, if the requestingUser is a member of the project these tasks are in.
// WARNING: This method, unfortunately, doesn't check the task relation to project, so there might be broken references
// left (from a project to a not existing task). So: USE WITH CARE!!!
// This relates to the github issue https://github.com/hauke96/simple-task-manager/issues/33
func (s *TaskService) Delete(taskIds []string, requestingUserId string) error {
	err := s.permissionService.VerifyMembershipTasks(taskIds, requestingUserId)
	if err != nil {
		return err
	}

	err = s.store.delete(taskIds)
	if err != nil {
		return err
	}

	return nil
}
