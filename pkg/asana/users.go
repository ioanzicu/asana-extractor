package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ioanzicu/asana-extractor/pkg/client"
)

const (
	baseURL = "https://app.asana.com/api/1.0"
)

// Client is the Asana API client
type Client struct {
	httpClient *client.Client
	workspace  string
}

// NewClient creates a new Asana API client
func NewClient(httpClient *client.Client, workspace string) *Client {
	return &Client{
		httpClient: httpClient,
		workspace:  workspace,
	}
}

// GetUsers retrieves users with pagination
func (c *Client) GetUsers(ctx context.Context, limit, offset int) ([]User, error) {
	// Build URL with query parameters
	u, err := url.Parse(fmt.Sprintf("%s/workspaces/%s/users", baseURL, c.workspace))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))
	q.Set("opt_fields", "gid,name,email,photo,workspaces")
	u.RawQuery = q.Encode()

	// Make request
	body, err := c.httpClient.GetBody(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Parse response
	var resp UsersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse users response: %w", err)
	}

	return resp.Data, nil
}

// GetAllUsers retrieves all users by automatically handling pagination
func (c *Client) GetAllUsers(ctx context.Context) ([]User, error) {
	const pageSize = 100 // to refactor
	var allUsers []User
	offset := 0

	for {
		users, err := c.GetUsers(ctx, pageSize, offset)
		if err != nil {
			return nil, err
		}

		if len(users) == 0 {
			break
		}

		allUsers = append(allUsers, users...)

		// If we got fewer results than the page size, we're done
		if len(users) < pageSize {
			break
		}

		offset += len(users)
	}

	return allUsers, nil
}
