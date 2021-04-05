import { WORKSPACE_ID } from "./index";

export default function configureRoleMapping(route) {
  route.get("/rolemapping", function (schema, request) {
    return schema.roleMappings.where((roleMapping) => {
      return roleMapping.workspaceId == WORKSPACE_ID;
    });
  });

  route.post("/rolemapping", function (schema, request) {
    const ts = Date.now();
    const attrs = {
      ...this.normalizedRequestAttrs("role-mapping"),
      workspaceId: WORKSPACE_ID,
    };
    const newRoleMapping = {
      ...attrs,
      createdTs: ts,
      lastUpdatedTs: ts,
      role: attrs.role,
      updaterId: attrs.updaterId,
      workspaceId: WORKSPACE_ID,
    };
    return schema.roleMappings.create(newRoleMapping);
  });

  route.patch("/rolemapping/:roleMappingId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("role-mapping");
    return schema.roleMappings.find(request.params.roleMappingId).update(attrs);
  });

  route.delete("/rolemapping/:roleMappingId", function (schema, request) {
    return schema.roleMappings.find(request.params.roleMappingId).destroy();
  });
}
