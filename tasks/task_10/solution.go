package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{clock: clock, tasks: map[string]Task{}}
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	r.mu.Lock()
	title = strings.TrimSpace(title)
	if title == "" {
		r.mu.Unlock()
		return Task{}, ErrInvalidTitle
	}
	var taskId = fmt.Sprintf("%d", r.seq)
	r.seq++

	var task = Task{ID: taskId, Title: title, Done: false, UpdatedAt: r.clock.Now()}
	r.tasks[taskId] = task
	r.mu.Unlock()

	return task, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.Lock()
	task, found := r.tasks[id]
	if !found {
		/* если был передан несуществующий id -> возвращаем пустую задачу и признак наличия в репозитории = false */
		r.mu.Unlock()
		return Task{}, false
	}
	r.mu.Unlock()

	return task, true
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	result := make([]Task, 0)

	r.mu.Lock()
	for _, task := range r.tasks {
		if done == task.Done {
			result = append(result, task)
		}
	}
	r.mu.Unlock()

	return result
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	task, found := r.tasks[id]
	if !found {
		/* если был передан несуществующий id -> возвращаем ошибку */
		r.mu.Unlock()
		return task, ErrNotFound
	}
	if done != task.Done {
		/* идемпотентно обновляем дату последней смены статуса */
		task.UpdatedAt = r.clock.Now()
		task.Done = done
		/* обновляем элемент по ключу, так как выше работаем с его копией */
		r.tasks[id] = task
	}
	r.mu.Unlock()

	return task, nil
}

type httpHandler struct{ repo TaskRepo }

type errorMessage struct {
	Message string `json:"message"`
	Error   error  `json:"error"`
}

var taskPathPattern = regexp.MustCompile(`^/tasks/([^/]+)$`)

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/tasks" {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else if matches := taskPathPattern.FindStringSubmatch(r.URL.Path); matches != nil {
		switch r.Method {
		case http.MethodGet:
			h.handleGet(w, r, matches[1])
		case http.MethodPatch:
			h.handlePatch(w, r, matches[1])
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

type createTaskRequest struct {
	Title *string `json:"title"`
}

func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body createTaskRequest
	if err := decodeStrictJSON(r.Body, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: "Invalid body", Error: err})
		return
	} else if body.Title == nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: "Parameter title must be not null", Error: err})
		return
	}

	task, err := h.repo.Create(*body.Title)
	if errors.Is(err, ErrInvalidTitle) {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: "Parameter title must be not empty", Error: err})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorMessage{Message: "Internal server error", Error: err})
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, found := h.repo.Get(id)
	if !found {
		writeJSON(w, http.StatusNotFound, errorMessage{Message: "Task not found"})
		return
	}
	writeJSON(w, 200, task)
}

func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	var done bool
	queryValues, exists := r.URL.Query()["done"]
	if !exists {
		/* считаем, что done по-умолчанию false */
		done = false
	} else {
		if len(queryValues) > 1 || queryValues[0] != "true" && queryValues[0] != "false" {
			writeJSON(w, http.StatusBadRequest, errorMessage{Message: "Invalid value for query parameter done"})
			return
		}
		/* по проверке выше в query-параметрах гарантированно был передан либо true, либо false */
		done, _ = strconv.ParseBool(queryValues[0])
	}

	var result = h.repo.List(done)
	sort.Slice(result, func(i, j int) bool {
		left := result[i]
		right := result[j]
		if left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.ID < right.ID
		}
		return left.UpdatedAt.After(right.UpdatedAt)
	})

	writeJSON(w, http.StatusOK, result)
}

type patchTaskRequest struct {
	Done *bool `json:"done"`
}

func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body patchTaskRequest
	if err := decodeStrictJSON(r.Body, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: "Invalid body", Error: err})
		return
	} else if body.Done == nil {
		writeJSON(w, http.StatusBadRequest, errorMessage{Message: "Parameter done must be not null", Error: err})
		return
	}

	task, err := h.repo.SetDone(id, *body.Done)
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorMessage{Message: fmt.Sprintf("Task with id=%s not found", id), Error: err})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorMessage{Message: "Internal server error", Error: err})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	/* запрещаем передачу неизвестных полей в body */
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	/* проверка, что в body передан один json объект */
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
