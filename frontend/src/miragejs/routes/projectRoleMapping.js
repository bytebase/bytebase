import { Response } from "miragejs";
import { WORKSPACE_ID, OWNER_ID } from "./index";
import { postMessageToOwnerAndDBA } from "../utils";

export default function configureProrjectRoleMapping(route) {
  route.get("/project/:projectId/rolemapping", function (schema, request) {
    const project = schema.projects.find(request.params.projectId);

    if (!project) {
      return new Response(
        404,
        {},
        {
          errors: "Project id " + request.params.projectId + " not found",
        }
      );
    }

    return schema.projectRoleMappings.where((roleMapping) => {
      return (
        roleMapping.workspaceId == WORKSPACE_ID &&
        roleMapping.projectId == request.params.projectId
      );
    });
  });

  route.post("/project/:projectId/rolemapping", function (schema, request) {
    const project = schema.projects.find(request.params.projectId);

    if (!project) {
      return new Response(
        404,
        {},
        {
          errors: "Project id " + request.params.projectId + " not found",
        }
      );
    }

    const ts = Date.now();
    const attrs = this.normalizedRequestAttrs("project-role-mapping");

    const roleMapping = schema.projectRoleMappings.findBy({
      principalId: attrs.principalId,
      projectId: request.params.projectId,
      workspaceId: WORKSPACE_ID,
    });
    if (roleMapping) {
      return roleMapping;
    }
    const newRoleMapping = {
      ...attrs,
      creatorId: attrs.creatorId,
      updaterId: attrs.creatorId,
      createdTs: ts,
      lastUpdatedTs: ts,
      role: attrs.role,
      principalId: attrs.principalId,
      projectId: request.params.projectId,
      workspaceId: WORKSPACE_ID,
    };

    const createdRoleMapping = schema.projectRoleMappings.create(
      newRoleMapping
    );

    const messageTemplate = {
      containerId: request.params.projectId,
      createdTs: ts,
      lastUpdatedTs: ts,
      type: "bb.msg.project.member.create",
      status: "DELIVERED",
      creatorId: attrs.creatorId,
      workspaceId: WORKSPACE_ID,
      payload: {
        principalId: attrs.principalId,
        newRole: attrs.role,
      },
    };
    postMessageToOwnerAndDBA(schema, attrs.updaterId, messageTemplate);

    return createdRoleMapping;
  });

  route.patch(
    "/project/:projectId/rolemapping/:roleMappingId",
    function (schema, request) {
      const project = schema.projects.find(request.params.projectId);

      if (!project) {
        return new Response(
          404,
          {},
          {
            errors: "Project id " + request.params.projectId + " not found",
          }
        );
      }

      const roleMapping = schema.projectRoleMappings.find(
        request.params.roleMappingId
      );
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

      const attrs = this.normalizedRequestAttrs("project-role-mapping");
      const updatedRoleMapping = roleMapping.update(attrs);

      const ts = Date.now();
      const messageTemplate = {
        containerId: request.params.projectId,
        createdTs: ts,
        lastUpdatedTs: ts,
        type: "bb.msg.project.member.updaterole",
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
    }
  );

  route.delete(
    "/project/:projectId/rolemapping/:roleMappingId",
    function (schema, request) {
      const project = schema.projects.find(request.params.projectId);

      if (!project) {
        return new Response(
          404,
          {},
          {
            errors: "Project id " + request.params.projectId + " not found",
          }
        );
      }

      const roleMapping = schema.projectRoleMappings.find(
        request.params.roleMappingId
      );
      if (!roleMapping) {
        return new Response(
          404,
          {},
          {
            errors:
              "Project role mapping id " +
              request.params.roleMappingId +
              " not found",
          }
        );
      }

      const oldRole = roleMapping.role;
      roleMapping.destroy();

      // NOTE, in actual implementation, we need to fetch the user from the auth context.
      const callerId = OWNER_ID;
      const ts = Date.now();
      const messageTemplate = {
        containerId: request.params.projectId,
        createdTs: ts,
        lastUpdatedTs: ts,
        type: "bb.msg.project.member.revoke",
        status: "DELIVERED",
        creatorId: callerId,
        workspaceId: WORKSPACE_ID,
        payload: {
          principalId: roleMapping.principalId,
          oldRole,
        },
      };
      postMessageToOwnerAndDBA(schema, callerId, messageTemplate);
    }
  );
}
