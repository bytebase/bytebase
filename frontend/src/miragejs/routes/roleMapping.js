import { Response } from "miragejs";
import { WORKSPACE_ID, OWNER_ID } from "./index";
import { postMessageToOwnerAndDBA } from "../utils";

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
    const roleMapping = schema.roleMappings.findBy({
      principalId: attrs.principalId,
      workspaceId: WORKSPACE_ID,
    });
    if (roleMapping) {
      return roleMapping;
    }
    const newRoleMapping = {
      ...attrs,
      createdTs: ts,
      lastUpdatedTs: ts,
      role: attrs.role,
      updaterId: attrs.updaterId,
      workspaceId: WORKSPACE_ID,
    };

    const createdRoleMapping = schema.roleMappings.create(newRoleMapping);

    const user = schema.users.find(attrs.principalId);
    const type =
      user.status == "INVITED"
        ? "bb.msg.member.invite"
        : "bb.msg.member.create";

    const messageTemplate = {
      containerId: WORKSPACE_ID,
      createdTs: ts,
      lastUpdatedTs: ts,
      type,
      status: "DELIVERED",
      creatorId: attrs.updaterId,
      workspaceId: WORKSPACE_ID,
      payload: {
        principalId: attrs.principalId,
        newRole: attrs.role,
      },
    };
    postMessageToOwnerAndDBA(schema, attrs.updaterId, messageTemplate);

    return createdRoleMapping;
  });

  route.patch("/rolemapping/:roleMappingId", function (schema, request) {
    const roleMapping = schema.roleMappings.find(request.params.roleMappingId);
    if (!roleMapping) {
      return new Response(
        404,
        {},
        {
          errors:
            "Role mapping id " + request.params.roleMappingId + " not found",
        }
      );
    }

    const oldRole = roleMapping.role;

    const attrs = this.normalizedRequestAttrs("role-mapping");
    const updatedRoleMapping = roleMapping.update(attrs);

    const ts = Date.now();
    const messageTemplate = {
      containerId: WORKSPACE_ID,
      createdTs: ts,
      lastUpdatedTs: ts,
      type: "bb.msg.member.updaterole",
      status: "DELIVERED",
      creatorId: attrs.updaterId,
      workspaceId: WORKSPACE_ID,
      payload: {
        principalId: roleMapping.principalId,
        oldRole,
        newRole: updatedRoleMapping.role,
      },
    };
    postMessageToOwnerAndDBA(schema, attrs.updaterId, messageTemplate);

    return updatedRoleMapping;
  });

  route.delete("/rolemapping/:roleMappingId", function (schema, request) {
    const roleMapping = schema.roleMappings.find(request.params.roleMappingId);

    if (!roleMapping) {
      return new Response(
        404,
        {},
        {
          errors:
            "Role mapping id " + request.params.roleMappingId + " not found",
        }
      );
    }

    const oldRole = roleMapping.role;
    roleMapping.destroy();

    // NOTE, in actual implementation, we need to fetch the user from the auth context.
    const callerId = OWNER_ID;
    const ts = Date.now();
    const messageTemplate = {
      containerId: WORKSPACE_ID,
      createdTs: ts,
      lastUpdatedTs: ts,
      type: "bb.msg.member.revoke",
      status: "DELIVERED",
      creatorId: callerId,
      workspaceId: WORKSPACE_ID,
      payload: {
        principalId: roleMapping.principalId,
        oldRole,
      },
    };
    postMessageToOwnerAndDBA(schema, callerId, messageTemplate);
  });
}
