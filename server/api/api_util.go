package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hauke96/sigolo"
	"github.com/hauke96/simple-task-manager/server/auth"
	"github.com/hauke96/simple-task-manager/server/database"
	"github.com/hauke96/simple-task-manager/server/permission"
	"github.com/hauke96/simple-task-manager/server/project"
	"github.com/hauke96/simple-task-manager/server/task"
	"github.com/hauke96/simple-task-manager/server/util"
	"github.com/pkg/errors"
	"net/http"
)

type ApiResponse struct {
	statusCode int
	data       interface{}
}

func BadRequestError(err error) *ApiResponse {
	return &ApiResponse{
		statusCode: http.StatusBadRequest,
		data:       err,
	}
}

func InternalServerError(err error) *ApiResponse {
	return &ApiResponse{
		statusCode: http.StatusInternalServerError,
		data:       err,
	}
}

func JsonResponse(data interface{}) *ApiResponse {
	return &ApiResponse{
		statusCode: http.StatusOK,
		data:       data,
	}
}

func EmptyResponse() *ApiResponse {
	return &ApiResponse{
		statusCode: http.StatusOK,
		data:       nil,
	}
}

type Context struct {
	token          *auth.Token
	transaction    *sql.Tx
	projectService *project.ProjectService
	taskService    *task.TaskService
}

func printRoutes(router *mux.Router) {
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		sigolo.Info("  %-*v %s", 7, methods, path)
		return nil
	})
}

func authenticatedTransactionHandler(handler func(r *http.Request, context *Context) *ApiResponse) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		prepareAndHandle(w, r, handler)
	}
}

func authenticatedWebsocket(handler func(w http.ResponseWriter, r *http.Request, token *auth.Token)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		t := query.Get("token")
		if t == "" || t == "null" || t == "\u009e" {
			err := errors.New("could not establish websocket connection: query parameter 'token' not set")
			util.ResponseUnauthorized(w, err)
			return
		}
		query.Del("token")

		// Add token query param value (set by websocket clients) as authorization so that verifyRequest can check it.
		r.Header.Add("Authorization", t)

		token, err := auth.VerifyRequest(r)
		if err != nil {
			sigolo.Error("Token verification failed: %s", err)
			// No further information to caller (which is a potential attacker)
			util.ResponseUnauthorized(w, errors.New("No valid authentication token found"))
			return
		}

		handler(w, r, token)
	}
}

// prepareAndHandle gets and verifies the token from the request, creates the context, starts a transaction, manages
// commit/rollback, calls the handler and also does error handling. When this function returns, everything should have a
// valid state: The response as well as the transaction (database).
func prepareAndHandle(w http.ResponseWriter, r *http.Request, handler func(r *http.Request, context *Context) *ApiResponse) {
	token, err := auth.VerifyRequest(r)
	if err != nil {
		sigolo.Debug("URL without valid token called: %s", r.URL.Path)
		sigolo.Error("Token verification failed: %s", err)
		// No further information to caller (which is a potential attacker)
		util.ResponseUnauthorized(w, errors.New("No valid authentication token found"))
		return
	}

	sigolo.Info("Call from '%s' (%s) to %s %s", token.User, token.UID, r.Method, r.URL.Path)

	// Create context with a new transaction and new service instances
	context, err := createContext(token)
	if err != nil {
		sigolo.Error("Unable to create context: %s", err)
		sigolo.Stack(err)
		// No further information to caller (which is a potential attacker)
		util.ResponseInternalError(w, errors.New("Unable to create context"))
		return
	}

	// Recover from panic and perform rollback on transaction
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch r := r.(type) {
			case error:
				err = r
			default:
				err = fmt.Errorf("%v", r)
			}

			sigolo.Error(fmt.Sprintf("!! PANIC !! Recover from panic:"))
			sigolo.Stack(err)

			util.ResponseInternalError(w, err)

			sigolo.Info("Try to perform rollback")
			rollbackErr := context.transaction.Rollback()
			if rollbackErr != nil {
				sigolo.Stack(errors.Wrap(rollbackErr, "error performing rollback"))
			}
		}
	}()

	// Call actual logic
	var response *ApiResponse
	response = handler(r, context)

	if response.statusCode != http.StatusOK {
		// Cause panic which will be recovered using the above function. This will then trigger a transaction rollback.
		panic(response.data.(error))
	}

	// Commit transaction
	err = context.transaction.Commit()
	if err != nil {
		sigolo.Error("Unable to commit transaction: %s", err.Error())
		panic(err)
	}
	sigolo.Debug("Committed transaction")

	if response.data != nil {
		encoder := json.NewEncoder(w)
		encoder.Encode(response.data)
	}
}

// createContext starts a new transaction and creates new service instances which use this new transaction so that all
// services (also those calling each other) are using the same transaction.
func createContext(token *auth.Token) (*Context, error) {
	context := &Context{}
	context.token = token

	tx, err := database.GetTransaction()
	if err != nil {
		return nil, errors.Wrap(err, "error getting transaction")
	}
	context.transaction = tx

	permissionService := permission.Init(tx)
	context.taskService = task.Init(tx, permissionService)
	context.projectService = project.Init(tx, context.taskService, permissionService)

	return context, nil
}
