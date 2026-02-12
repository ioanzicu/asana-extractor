package asana

import "time"

// User represents an Asana user
type User struct {
	GID          string      `json:"gid"`
	ResourceType string      `json:"resource_type"`
	Name         string      `json:"name"`
	Email        string      `json:"email,omitempty"` // used?
	Photo        *Photo      `json:"photo,omitempty"`
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

// Photo represents a user's photo
type Photo struct {
	Image21x21   string `json:"image_21x21,omitempty"`
	Image27x27   string `json:"image_27x27,omitempty"`
	Image36x36   string `json:"image_36x36,omitempty"`
	Image60x60   string `json:"image_60x60,omitempty"`
	Image128x128 string `json:"image_128x128,omitempty"`
}

// Response wraps API responses
type Response struct {
	Data interface{} `json:"data"`
}

// UsersResponse wraps the users list response
type UsersResponse struct {
	Data []User `json:"data"`
}

// ProjectsResponse wraps the projects list response
type ProjectsResponse struct {
	Data     []Project `json:"data"`
	NextPage *NextPage `json:"next_page"`
}

// NextPage represents the pagination information
type NextPage struct {
	Offset string `json:"offset"`
	Path   string `json:"path"`
	Uri    string `json:"uri"`
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
