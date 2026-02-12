package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ioanzicu/asana-extractor/pkg/client"
)

// Client is the Asana API client
type Client struct {
	httpClient   *client.Client
	workspace    string
	baseURL      string
	userPageSize int
}

// NewClient creates a new Asana API client
func NewClient(httpClient *client.Client, workspace string, baseURL string, userPageSize int) *Client {
	return &Client{
		httpClient:   httpClient,
		workspace:    workspace,
		baseURL:      baseURL,
		userPageSize: userPageSize,
	}
}

// GetUsers retrieves users with pagination
func (c *Client) GetUsers(ctx context.Context, limit int, offset string) ([]User, *NextPage, error) {
	// Build URL with query parameters
	u, err := url.Parse(fmt.Sprintf("%s/workspaces/%s/users", c.baseURL, c.workspace))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))

	if offset != "" {
		q.Set("offset", offset)
	}
	q.Set("opt_fields", "gid,name,email,workspaces")
	u.RawQuery = q.Encode()

	// Make request
	body, err := c.httpClient.GetBody(ctx, u.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Parse response
	var resp UsersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse users response: %w", err)
	}

	return resp.Data, resp.NextPage, nil
}

// GetAllUsers retrieves all users by automatically handling pagination
func (c *Client) GetAllUsers(ctx context.Context) ([]User, error) {
	var allUsers []User
	var currentOffset string

	for {
		users, nextPage, err := c.GetUsers(ctx, c.userPageSize, currentOffset)
		if err != nil {
			return nil, err
		}

		if len(users) == 0 {
			break
		}

		allUsers = append(allUsers, users...)

		// If we got fewer results than the page size, we're done
		if nextPage == nil || nextPage.Offset == "" {
			break
		}

		currentOffset = nextPage.Offset
	}

	return allUsers, nil
}
