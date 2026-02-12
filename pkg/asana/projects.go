package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetProjects retrieves projects with pagination
func (c *Client) GetProjects(ctx context.Context, limit int, offset string) ([]Project, *NextPage, error) {
	// Build URL with query parameters
	u, err := url.Parse(fmt.Sprintf("%s/workspaces/%s/projects", c.baseURL, c.workspace))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))

	if offset != "" {
		q.Set("offset", offset)
	}

	q.Set("opt_fields", "gid,name,archived,color,created_at,modified_at,owner,public,workspace,team")
	u.RawQuery = q.Encode()

	// Make request
	body, err := c.httpClient.GetBody(ctx, u.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get projects: %w", err)
	}

	// Parse response
	var resp ProjectsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse projects response: %w", err)
	}

	return resp.Data, resp.NextPage, nil
}

// GetAllProjects retrieves all projects by automatically handling pagination
func (c *Client) GetAllProjects(ctx context.Context) ([]Project, error) {
	const pageSize = 100
	var allProjects []Project
	var currentOffset string

	for {
		projects, nextPage, err := c.GetProjects(ctx, pageSize, currentOffset)
		if err != nil {
			return nil, err
		}

		if len(projects) == 0 {
			break
		}

		allProjects = append(allProjects, projects...)

		if nextPage == nil || nextPage.Offset == "" {
			break
		}

		currentOffset = nextPage.Offset
	}

	return allProjects, nil
}
