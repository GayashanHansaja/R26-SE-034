package idp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type API struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Method      string    `json:"method"`
	AuthType    string    `json:"authType"`
	AuthHeader  string    `json:"authHeader,omitempty"`
	AuthKey     string    `json:"authKey,omitempty"`
	AuthUsername string   `json:"authUsername,omitempty"`
	AuthToken   string    `json:"authToken,omitempty"`
	Module      string    `json:"module"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Registry struct {
	path string
	APIs map[string]API `json:"apis"`
}

func NewRegistry(path string) (*Registry, error) {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".bridgectl", "registry.json")
	}

	reg := &Registry{
		path: path,
		APIs: make(map[string]API),
	}

	if err := reg.load(); err != nil {
		// If file doesn't exist, it's fine, we'll create it on save
	}

	return reg, nil
}

func (r *Registry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, r)
}

func (r *Registry) save() error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}

func (r *Registry) Register(api *API) error {
	if api.ID == "" {
		api.ID = fmt.Sprintf("api-%d", time.Now().UnixNano())
	}
	api.CreatedAt = time.Now()
	api.Status = "active"
	r.APIs[api.Name] = *api
	return r.save()
}

func (r *Registry) List() []API {
	list := make([]API, 0, len(r.APIs))
	for _, api := range r.APIs {
		list = append(list, api)
	}
	return list
}

func (r *Registry) Delete(name string) error {
	delete(r.APIs, name)
	return r.save()
}

func (r *Registry) Get(name string) (API, bool) {
	api, ok := r.APIs[name]
	return api, ok
}
