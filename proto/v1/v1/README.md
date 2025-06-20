# Proto Service Permission Comment Convention

## Purpose

To ensure that required permissions for each API endpoint are clearly documented and included in the generated OpenAPI documentation, we follow a strict convention for annotating RPCs in all service `.proto` files in this directory.

## Convention

- **For every `rpc` operation**, add a comment immediately above the `rpc` definition indicating the required permissions.
- The comment must be of the form:
  - `// Permissions required: None` (if no permission is required)
  - `// Permissions required: bb.permission.name` (if one permission is required, with the `bb.` prefix included)
  - `// Permissions required: bb.permission.one, bb.permission.two` (if multiple permissions are required, all listed, `bb.` prefix included)
- The comment must be placed **immediately above the `rpc` line** so that OpenAPI generators (such as gnostic) include it in the endpoint's description.

## Example

```proto
// Permissions required: bb.projects.get
rpc ListProjects(ListProjectsRequest) returns (ListProjectsResponse) {
  option (bytebase.v1.permission) = "bb.projects.get";
  // ...
}

// Permissions required: None
rpc GetStatus(GetStatusRequest) returns (StatusResponse) {
  // ...
}

// Permissions required: bb.projects.update, bb.projects.delete
rpc UpdateOrDelete(UpdateOrDeleteRequest) returns (UpdateOrDeleteResponse) {
  option (bytebase.v1.permission) = "bb.projects.update,bb.projects.delete";
  // ...
}
```

## Why?

- This ensures that permission requirements are visible to both developers and API consumers.
- The comments are automatically included in the OpenAPI `description` field for each endpoint when using gnostic.
- It helps maintain consistency and clarity as the API evolves.

## Maintenance

- When adding or modifying an `rpc`, always update or add the permission comment accordingly.
- If you add a new service file, ensure this convention is followed for all its RPCs.
- If you change permission requirements, update the comment to match.

## Questions?

Contact the maintainers or check with the team if you are unsure about the correct permissions for an endpoint. 