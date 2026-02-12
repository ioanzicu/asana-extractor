package asana

import "time"

// User represents an Asana user
type User struct {
	GID          string      `json:"gid"`
	ResourceType string      `json:"resource_type"`
	Name         string      `json:"name"`
	Email        string      `json:"email,omitempty"`
	Workspaces   []Workspace `json:"workspaces,omitempty"`
}

// Project represents an Asana project
type Project struct {
	GID          string     `json:"gid"`
	ResourceType string     `json:"resource_type"`
	Name         string     `json:"name"`
	Archived     bool       `json:"archived"`
	Color        string     `json:"color,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ModifiedAt   time.Time  `json:"modified_at"`
	Owner        *User      `json:"owner,omitempty"`
	Public       bool       `json:"public"`
	Workspace    *Workspace `json:"workspace,omitempty"`
	Team         *Team      `json:"team,omitempty"`
}

// Workspace represents an Asana workspace
type Workspace struct {
	GID          string `json:"gid"`
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
}

// Team represents an Asana team
type Team struct {
	GID          string `json:"gid"`
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
}

// Response wraps API responses
type Response struct {
	Data any `json:"data"`
}

// UsersResponse wraps the users list response
type UsersResponse struct {
	Data     []User    `json:"data"`
	NextPage *NextPage `json:"next_page"`
}

type NextPage struct {
	Offset string `json:"offset"`
	Path   string `json:"path"`
	Uri    string `json:"uri"`
}

// ProjectsResponse wraps the projects list response
type ProjectsResponse struct {
	Data     []Project `json:"data"`
	NextPage *NextPage `json:"next_page"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

// Error represents a single error in the response
type Error struct {
	Message string `json:"message"`
	Help    string `json:"help,omitempty"`
}
