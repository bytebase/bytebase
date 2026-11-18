package neon

import (
	"context"

	"encoding/base64"
	"encoding/json"
	"net/url"

	"github.com/pkg/errors"

	"golang.org/x/sync/errgroup"
)

// DiscoveryPage contains only candidates inspected by this request.
type DiscoveryPage struct {
	Branches      []DiscoveredBranch
	Failures      []ProjectFailure
	NextPageToken string
}
type ProjectFailure struct {
	ProjectID string
	Error     error
}

// Offsets refer to fixed-size provider pages, so clients may change page size.
// No credentials or discovered resource contents are retained in the cursor.
type discoveryCursor struct {
	Version       int    `json:"v"`
	Filter        string `json:"f,omitempty"`
	Organization  int    `json:"o,omitempty"`
	Projects      string `json:"p,omitempty"`
	ProjectOffset int    `json:"i,omitempty"`
	Branches      string `json:"b,omitempty"`
	BranchOffset  int    `json:"j,omitempty"`
}

func sharedFailure(err error) bool {
	var api *APIError
	if errors.As(err, &api) {
		return api.StatusCode == 401 || api.StatusCode == 429 || api.StatusCode >= 500
	}
	return !errors.Is(err, ErrEndpointUnavailable)
}

// DiscoverPage traverses at most ten provider branch/project pages per request.
// An empty page can have a continuation token when traversal still has work.
func (c *Client) DiscoverPage(ctx context.Context, projectID string, pageSize int, token string) (*DiscoveryPage, error) {
	if projectID != "" {
		if err := validIDs(projectID); err != nil {
			return nil, err
		}
	}
	if pageSize < 0 {
		return nil, errors.Wrap(ErrInvalidArgument, "page size must not be negative")
	}
	if pageSize == 0 {
		pageSize = 50
	}
	pageSize = min(pageSize, 100)
	cursor := discoveryCursor{Version: 1, Filter: projectID}
	if token != "" {
		if len(token) > 16384 {
			return nil, ErrInvalidPageToken
		}
		data, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || json.Unmarshal(data, &cursor) != nil || cursor.Version != 1 || cursor.Filter != projectID || cursor.Organization < 0 || cursor.ProjectOffset < 0 || cursor.ProjectOffset > 100 || cursor.BranchOffset < 0 || cursor.BranchOffset > 100 {
			return nil, ErrInvalidPageToken
		}
		if projectID != "" && (cursor.Organization != 0 || cursor.Projects != "" || cursor.ProjectOffset != 0) {
			return nil, ErrInvalidPageToken
		}
	}
	result := &DiscoveryPage{}
	var organizations []Organization
	var scoped Project
	if projectID != "" {
		var response struct {
			Project Project `json:"project"`
		}
		if err := c.get(ctx, "/projects/"+projectID, nil, &response); err != nil {
			if sharedFailure(err) {
				return nil, err
			}
			result.Failures = append(result.Failures, ProjectFailure{ProjectID: projectID, Error: err})
			return result, nil
		}
		if response.Project.ID != projectID || response.Project.Name == "" {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon project response")
		}
		scoped = response.Project
	} else {
		var err error
		organizations, err = c.ListOrganizations(ctx)
		if err != nil {
			return nil, err
		}
		if len(organizations) == 0 {
			organizations = []Organization{{}}
		}
	}
	nextProject := func() { cursor.ProjectOffset++; cursor.Branches = ""; cursor.BranchOffset = 0 }
	for range 10 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		project := scoped
		if projectID == "" {
			if cursor.Organization >= len(organizations) {
				return result, nil
			}
			query := url.Values{"limit": {"100"}}
			if id := organizations[cursor.Organization].ID; id != "" {
				query.Set("org_id", id)
			}
			if cursor.Projects != "" {
				query.Set("cursor", cursor.Projects)
			}
			var response struct {
				Projects    []Project `json:"projects"`
				Unavailable []string  `json:"unavailable_project_ids"`
				Pagination  struct {
					Cursor string `json:"cursor"`
				} `json:"pagination"`
			}
			if err := c.get(ctx, "/projects", query, &response); err != nil {
				return nil, err
			}
			if response.Projects == nil || len(response.Projects) > 100 {
				return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon projects response")
			}
			// The cursor holds this position only on the first visit to a provider
			// page, so each unavailable project is reported once per traversal.
			if cursor.ProjectOffset == 0 && cursor.Branches == "" && cursor.BranchOffset == 0 {
				for _, id := range response.Unavailable {
					if validIDs(id) != nil {
						return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon project")
					}
					result.Failures = append(result.Failures, ProjectFailure{ProjectID: id, Error: ErrUnavailable})
				}
			}
			if cursor.ProjectOffset >= len(response.Projects) {
				if len(response.Projects) > 0 && response.Pagination.Cursor != "" {
					if response.Pagination.Cursor == cursor.Projects {
						return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon pagination cursor")
					}
					cursor.Projects = response.Pagination.Cursor
				} else {
					cursor.Organization++
					cursor.Projects = ""
				}
				cursor.ProjectOffset = 0
				cursor.Branches = ""
				cursor.BranchOffset = 0
				continue
			}
			project = response.Projects[cursor.ProjectOffset]
			if validIDs(project.ID) != nil || project.Name == "" {
				return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon project")
			}
		}
		query := url.Values{"limit": {"100"}, "sort_by": {"created_at"}, "sort_order": {"asc"}}
		if cursor.Branches != "" {
			query.Set("cursor", cursor.Branches)
		}
		var response struct {
			Branches   []Branch `json:"branches"`
			Pagination struct {
				Next string `json:"next"`
			} `json:"pagination"`
		}
		if err := c.get(ctx, "/projects/"+project.ID+"/branches", query, &response); err != nil {
			if sharedFailure(err) {
				return nil, err
			}
			result.Failures = append(result.Failures, ProjectFailure{ProjectID: project.ID, Error: err})
			if projectID != "" {
				return result, nil
			}
			nextProject()
			continue
		}
		if response.Branches == nil || len(response.Branches) > 100 {
			return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon branches response")
		}
		end := min(len(response.Branches), cursor.BranchOffset+pageSize-len(result.Branches))
		if end > cursor.BranchOffset {
			branches, err := c.enrichBranches(ctx, project, response.Branches[cursor.BranchOffset:end])
			if err != nil {
				return nil, err
			}
			result.Branches = append(result.Branches, branches...)
			cursor.BranchOffset = end
		}
		if cursor.BranchOffset >= len(response.Branches) {
			cursor.BranchOffset = 0
			if response.Pagination.Next != "" {
				if response.Pagination.Next == cursor.Branches {
					return nil, errors.Wrap(ErrInvalidResponse, "invalid Neon pagination cursor")
				}
				cursor.Branches = response.Pagination.Next
			} else {
				if projectID != "" {
					return result, nil
				}
				nextProject()
			}
		}
		if len(result.Branches) >= pageSize {
			break
		}
	}
	data, err := json.Marshal(cursor)
	if err != nil {
		return nil, err
	}
	result.NextPageToken = base64.RawURLEncoding.EncodeToString(data)
	return result, nil
}

func (c *Client) enrichBranches(ctx context.Context, project Project, branches []Branch) ([]DiscoveredBranch, error) {
	result := make([]DiscoveredBranch, len(branches))
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(4)
	for i, branch := range branches {
		group.Go(func() error {
			if validIDs(branch.ID) != nil || branch.Name == "" {
				return errors.Wrap(ErrInvalidResponse, "invalid Neon branch")
			}
			discovered := DiscoveredBranch{ProjectID: project.ID, ProjectName: project.Name, ID: branch.ID, Name: branch.Name}
			var err error
			discovered.Databases, err = c.ListDatabases(ctx, project.ID, branch.ID)
			if err == nil {
				discovered.Roles, err = c.ListRoles(ctx, project.ID, branch.ID)
			}
			if err == nil {
				var endpoints []Endpoint
				endpoints, err = c.ListEndpoints(ctx, project.ID, branch.ID)
				if err == nil {
					_, err = directEndpoint(endpoints, branch.ID)
					discovered.EndpointAvailable = err == nil
				}
			}
			if err != nil {
				if sharedFailure(err) {
					return err
				}
				discovered.Error = err
			}
			result[i] = discovered
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}
