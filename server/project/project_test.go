package project

import (
	"database/sql"
	"fmt"
	"github.com/hauke96/sigolo"
	"github.com/hauke96/simple-task-manager/server/config"
	"github.com/hauke96/simple-task-manager/server/database"
	"github.com/hauke96/simple-task-manager/server/permission"
	"github.com/hauke96/simple-task-manager/server/task"
	testHelper "github.com/hauke96/simple-task-manager/server/test"
	"github.com/pkg/errors"
	"testing"

	_ "github.com/lib/pq" // Make driver "postgres" usable
)

var (
	tx          *sql.Tx
	s           *ProjectService
	taskService *task.TaskService
)

func setup() {
	config.LoadConfig("../config/test.json")
	testHelper.InitWithDummyData()
	sigolo.LogLevel = sigolo.LOG_DEBUG

	var err error
	tx, err = database.GetTransaction()
	if err != nil {
		panic(err)
	}
	permissionService := permission.Init(tx)
	taskService = task.Init(tx, permissionService)
	s = Init(tx, taskService, permissionService)
}

func run(t *testing.T, testFunc func() error) {
	setup()

	err := testFunc()
	if err != nil {
		t.Errorf("%+v", err)
		t.Fail()
	}

	tearDown()
}

func tearDown() {
	err := tx.Commit()
	if err != nil {
		panic(err)
	}
}

func TestGetProjects(t *testing.T) {
	run(t, func() error {
		// For Maria (being part of project 1 and 2)
		userProjects, err := s.GetProjects("Maria")
		if err != nil {
			return err
		}
		if !contains("1", userProjects) {
			return errors.New("Maria is in deed project 1")
		}
		if !contains("2", userProjects) {
			return errors.New("Maria is in deed project 2")
		}
		if userProjects[0].TotalProcessPoints != 10 || userProjects[0].DoneProcessPoints != 0 {
			return errors.New("Process points on project not set correctly")
		}
		if userProjects[1].TotalProcessPoints != 308 || userProjects[1].DoneProcessPoints != 154 {
			return errors.New("Process points on project not set correctly")
		}

		// For Peter (being part of only project 1)
		userProjects, err = s.GetProjects("Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("Getting should work: %+v", err))
		}
		if !contains("1", userProjects) {
			return errors.New("Peter is in deed project 1")
		}
		if contains("2", userProjects) {
			return errors.New("Peter is not in project 2")
		}
		if userProjects[0].TotalProcessPoints != 10 || userProjects[0].DoneProcessPoints != 0 {
			return errors.New("Process points on project not set correctly")
		}
		return nil
	})
}

func TestGetProjectByTask(t *testing.T) {
	run(t, func() error {
		project, err := s.GetProjectByTask("4", "John")
		if err != nil {
			return err
		}

		if project.Id != "2" {
			return errors.New("Project ID not matching")
		}
		if project.TotalProcessPoints != 308 || project.DoneProcessPoints != 154 {
			return errors.New("Process points on project not set correctly")
		}
		return nil
	})
}

func TestGetTasks(t *testing.T) {
	run(t, func() error {
		tasks, err := s.GetTasks("1", "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("Get should work: %s", err.Error()))
		}

		sigolo.Debug("Tasks: %#v", tasks)

		if len(tasks) != 1 {
			return errors.New("There should be exactly one task")
		}

		task := tasks[0]
		sigolo.Debug("Task: %#v", task)

		if task.Id != "1" {
			return errors.New("id not matching")
		}

		if task.ProcessPoints != 0 {
			return errors.New("process points not matching")
		}

		if task.MaxProcessPoints != 10 {
			return errors.New("max process points not matching")
		}

		if task.AssignedUser != "Peter" {
			return errors.New("assigned user not matching")
		}

		// Part of project but not owning
		_, err = s.GetTasks("1", "Maria")
		if err != nil {
			return errors.New("This should work, Maria is part of the project")
		}

		// Not part of project
		_, err = s.GetTasks("1", "Unknown user")
		if err == nil {
			return errors.New("Get tasks of not owned project should not work")
		}

		// Not existing project
		_, err = s.GetTasks("28745276", "Peter")
		if err == nil {
			return errors.New("Get should not work")
		}
		return nil
	})
}

func TestAddAndGetProject(t *testing.T) {
	run(t, func() error {
		user := "Jack"
		p := Project{
			Name:    "Test name",
			TaskIDs: []string{"8"},
			Users:   []string{user, "user2"},
			Owner:   user,
		}

		newProject, err := s.AddProject(&p)
		if err != nil {
			return errors.New(fmt.Sprintf("Adding should work: %s", err.Error()))
		}

		if len(newProject.Users) != 2 {
			return errors.New(fmt.Sprintf("User amount should be 2 but was %d", len(newProject.Users)))
		}
		if newProject.Users[0] != user || newProject.Users[1] != "user2" {
			return errors.New(fmt.Sprintf("User not matching"))
		}
		if len(newProject.TaskIDs) != len(p.TaskIDs) || newProject.TaskIDs[0] != p.TaskIDs[0] {
			return errors.New(fmt.Sprintf("Task ID should be '%s' but was '%s'", newProject.TaskIDs[0], p.TaskIDs[0]))
		}
		if newProject.Name != p.Name {
			return errors.New(fmt.Sprintf("Name should be '%s' but was '%s'", newProject.Name, p.Name))
		}
		if newProject.Owner != user {
			return errors.New(fmt.Sprintf("Owner should be '%s' but was '%s'", user, newProject.Owner))
		}
		if newProject.TotalProcessPoints != 100 || newProject.DoneProcessPoints != 5 {
			return errors.New(fmt.Sprintf("Process points on project not set correctly"))
		}
		return nil
	})
}

func TestAddProjectWithUsedTasks(t *testing.T) {
	run(t, func() error {
		user := "Jen"
		p := Project{
			Name:    "Test name",
			TaskIDs: []string{"1", "22", "33"}, // one task already used in a project
			Users:   []string{user, "user2"},
			Owner:   user,
		}

		_, err := s.AddProject(&p)
		if err == nil {
			return errors.New(fmt.Sprintf("The tasks are already used. This should not work."))
		}
		return nil
	})
}

func TestAddUser(t *testing.T) {
	run(t, func() error {
		newUser := "new user"

		p, err := s.AddUser("1", newUser, "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("This should work: %s", err.Error()))
		}

		containsUser := false
		for _, u := range p.Users {
			if u == newUser {
				containsUser = true
				break
			}
		}
		if !containsUser {
			return errors.New("Project should contain new user")
		}
		if p.TotalProcessPoints != 10 || p.DoneProcessPoints != 0 {
			return errors.New(fmt.Sprintf("Process points on project not set correctly"))
		}

		p, err = s.AddUser("2284527", newUser, "Peter")
		if err == nil {
			return errors.New("This should not work: The project does not exist")
		}

		p, err = s.AddUser("1", newUser, "Not-Owning-User")
		if err == nil {
			return errors.New("This should not work: A non-owner user tries to add a user")
		}
		return nil
	})
}

func TestAddUserTwice(t *testing.T) {
	run(t, func() error {
		newUser := "another-new-user"

		_, err := s.AddUser("1", newUser, "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("This should work: %s", err.Error()))
		}

		// Add second time, this should now work
		_, err = s.AddUser("1", newUser, "Peter")
		if err == nil {
			return errors.New("Adding a user twice should not work")
		}
		return nil
	})
}

func TestRemoveUser(t *testing.T) {
	run(t, func() error {
		userToRemove := "Maria"

		p, err := s.RemoveUser("1", "Peter", userToRemove)
		if err != nil {
			return errors.New(fmt.Sprintf("This should work: %s", err.Error()))
		}

		containsUser := false
		for _, u := range p.Users {
			if u == userToRemove {
				containsUser = true
				break
			}
		}
		if containsUser {
			return errors.New("Project should not contain user anymore")
		}
		if p.TotalProcessPoints != 10 || p.DoneProcessPoints != 0 {
			return errors.New(fmt.Sprintf("Process points on project not set correctly"))
		}

		tasks, err := taskService.GetTasks(p.TaskIDs, "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("Getting tasks should still work"))
		}

		// Check that the user to remove has been unassigned
		for _, task := range tasks {
			if task.AssignedUser == userToRemove {
				return errors.New(fmt.Sprintf("Task '%s' still has user '%s' assigned", task.Id, userToRemove))
			}
		}

		// Not existing project
		p, err = s.RemoveUser("2284527", "Peter", userToRemove)
		if err == nil {
			return errors.New("This should not work: The project does not exist")
		}

		// Not owning user requesting removal
		p, err = s.RemoveUser("1", "Not-Owning-User", userToRemove)
		if err == nil {
			return errors.New("This should not work: A non-owner user requests removal")
		}
		return nil
	})
}

func TestRemoveNonOwnerUser(t *testing.T) {
	run(t, func() error {
		userToRemove := "Carl"

		// Carl is not owner and removes himself, which is ok
		p, err := s.RemoveUser("2", "Carl", userToRemove)
		if err != nil {
			return errors.New(fmt.Sprintf("This should work: %s", err.Error()))
		}

		containsUser := false
		for _, u := range p.Users {
			if u == userToRemove {
				containsUser = true
				break
			}
		}
		if containsUser {
			return errors.New("Project should not contain user anymore")
		}
		return nil
	})
}

func TestRemoveArbitraryUserNotAllowed(t *testing.T) {
	run(t, func() error {
		userToRemove := "Anna"

		// Michael is not member of the project and should not be allowed to remove anyone
		p, err := s.RemoveUser("2", "Michael", userToRemove)
		if err == nil {
			return errors.New(fmt.Sprintf("This should not work: %s", err.Error()))
		}

		p, err = s.GetProject("2", "Maria")
		if err != nil {
			return err
		}

		containsUser := false
		for _, u := range p.Users {
			if u == userToRemove {
				containsUser = true
				break
			}
		}
		if !containsUser {
			return errors.New("Project should still contain user")
		}

		// Remove not-member user:

		userToRemove = "Nina" // Not a member of the project
		p, err = s.RemoveUser("2", "Peter", userToRemove)
		if err == nil {
			return errors.New(fmt.Sprintf("This should not work: %s", err.Error()))
		}
		return nil
	})
}

func TestRemoveUserTwice(t *testing.T) {
	run(t, func() error {
		_, err := s.RemoveUser("2", "Maria", "John")
		if err != nil {
			t.Error("This should work: ", err)
		}

		// "John" was removed above to we remove him here the second time
		_, err = s.RemoveUser("2", "Maria", "John")
		if err == nil {
			return errors.New("Removing a user twice should not work")
		}
		return nil
	})
}

func TestLeaveProject(t *testing.T) {
	run(t, func() error {
		userToRemove := "Anna"

		p, err := s.LeaveProject("2", userToRemove)
		if err != nil {
			return errors.New(fmt.Sprintf("This should work: %s", err.Error()))
		}

		containsUser := false
		for _, u := range p.Users {
			if u == userToRemove {
				containsUser = true
				break
			}
		}
		if containsUser {
			return errors.New("Project should not contain user anymore")
		}
		if p.TotalProcessPoints != 308 || p.DoneProcessPoints != 154 {
			return errors.New(fmt.Sprintf("Process points on project not set correctly"))
		}

		// Owner should not be allowed to leave
		p, err = s.LeaveProject("2", "Maria")
		if err == nil {
			return errors.New("This should not work: The owner is not allowed to leave")
		}

		// Invalid project id
		p, err = s.LeaveProject("2284527", "Peter")
		if err == nil {
			return errors.New("This should not work: The project does not exist")
		}

		// Not existing user wants to leave
		p, err = s.LeaveProject("1", "Not-Existing-User")
		if err == nil {
			return errors.New("This should not work: A non-existing user should be removed")
		}

		// "Maria" was removed above to we remove her here the second time
		_, err = s.LeaveProject("2", userToRemove)
		if err == nil {
			return errors.New("Leaving a project twice should not work")
		}
		return nil
	})
}

func TestDeleteProject(t *testing.T) {
	run(t, func() error {
		id := "1" // owned by "Peter"

		// Try to remove with now-owning user

		err := s.DeleteProject(id, "Maria") // Maria does not own this project
		if err == nil {
			return errors.New("Maria does not own this project, this should not work")
		}

		_, err = s.GetProject(id, "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("The project should exist: %s", err.Error()))
		}

		// Actually remove project

		project, err := s.GetProject(id, "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("Error getting project to relete: %s", err.Error()))
		}
		taskIds := project.TaskIDs

		err = s.DeleteProject(id, "Peter") // Maria does not own this project
		if err != nil {
			return errors.New(fmt.Sprintf("Peter owns this project, this should work: %s", err.Error()))
		}

		_, err = s.GetProject(id, "Peter")
		if err == nil {
			return errors.New("The project should not exist anymore")
		}

		_, err = taskService.GetTasks(taskIds, "Peter")
		if err == nil {
			return errors.New("The tasks should not exist anymore")
		}

		// Delete not existing project

		err = s.DeleteProject("45356475", "Peter")
		if err == nil {
			return errors.New("This project does not exist, this should not work")
		}
		return nil
	})
}

func TestUpdateName(t *testing.T) {
	run(t, func() error {
		oldProject, err := s.GetProject("1", "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("Error getting project to update: %s", err))
		}

		newName := "flubby dubby"
		project, err := s.UpdateName("1", newName, "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("Error updating name wasn't expected: %s", err))
		}
		if project.Name != newName {
			return errors.New(fmt.Sprintf("New name doesn't match with expected one: %s != %s", oldProject.Name, newName))
		}
		if project.TotalProcessPoints != 10 || project.DoneProcessPoints != 0 {
			return errors.New(fmt.Sprintf("Process points on project not set correctly"))
		}

		// With newline

		newNewlineName := "foo\nbar\nwhatever"
		project, err = s.UpdateName("1", newNewlineName, "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("Error updating name wasn't expected: %s", err))
		}
		if project.Name != "foo" {
			return errors.New(fmt.Sprintf("New name doesn't match with expected one: %s != foo", oldProject.Name))
		}

		// With non-owner (Maria)

		_, err = s.UpdateName("1", "skfgkf", "Maria")
		if err == nil {
			return errors.New("Updating name should not be possible for non-owner user Maria")
		}

		// Empty name

		_, err = s.UpdateName("1", "  ", "Peter")
		if err == nil {
			return errors.New("Updating name should not be possible with empty name")
		}
		return nil
	})
}

func TestUpdateDescription(t *testing.T) {
	run(t, func() error {
		oldProject, _ := s.GetProject("1", "Peter")

		newDescription := "flubby dubby\n foo bar"
		project, err := s.UpdateDescription("1", newDescription, "Peter")
		if err != nil {
			return errors.New(fmt.Sprintf("Error updating description wasn't expected: %s", err))
		}
		if project.Description != newDescription {
			return errors.New(fmt.Sprintf("New description doesn't match with expected one: %s != %s", oldProject.Name, newDescription))
		}
		if project.TotalProcessPoints != 10 || project.DoneProcessPoints != 0 {
			return errors.New(fmt.Sprintf("Process points on project not set correctly"))
		}

		// With non-owner (Maria)

		_, err = s.UpdateDescription("1", "skfgkf", "Maria")
		if err == nil {
			return errors.New("Updating description should not be possible for non-owner user Maria")
		}

		// Empty description

		_, err = s.UpdateDescription("1", "  ", "Peter")
		if err == nil {
			return errors.New("Updating description should not be possible with empty description")
		}
		return nil
	})
}

func contains(projectIdToFind string, projectsToCheck []*Project) bool {
	for _, p := range projectsToCheck {
		if p.Id == projectIdToFind {
			return true
		}
	}

	return false
}
