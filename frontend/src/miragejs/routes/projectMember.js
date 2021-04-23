import { Response } from "miragejs";
import { WORKSPACE_ID, FAKE_API_CALLER_ID } from "./index";
import { postMessageToOwnerAndDBA } from "../utils";

export default function configureProrjectMember(route) {
  route.get("/project/:projectId/member", function (schema, request) {
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

    return schema.projectMembers.where((member) => {
      return (
        member.workspaceId == WORKSPACE_ID &&
        member.projectId == request.params.projectId
      );
    });
  });

  route.post("/project/:projectId/member", function (schema, request) {
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
    const attrs = this.normalizedRequestAttrs("project-member-new");

    const member = schema.projectMembers.findBy({
      principalId: attrs.principalId,
      projectId: request.params.projectId,
      workspaceId: WORKSPACE_ID,
    });
    if (member) {
      return member;
    }

    const newMember = {
      ...attrs,
      creatorId: attrs.creatorId,
      updaterId: attrs.creatorId,
      createdTs: ts,
      updatedTs: ts,
      role: attrs.role,
      principalId: attrs.principalId,
      projectId: request.params.projectId,
      workspaceId: WORKSPACE_ID,
    };

    const createdMember = schema.projectMembers.create(newMember);

    const messageTemplate = {
      containerId: request.params.projectId,
      createdTs: ts,
      updatedTs: ts,
      type: "bb.message.project.member.create",
      status: "DELIVERED",
      creatorId: attrs.creatorId,
      workspaceId: WORKSPACE_ID,
      payload: {
        principalId: attrs.principalId,
        newRole: attrs.role,
      },
    };
    postMessageToOwnerAndDBA(schema, attrs.creatorId, messageTemplate);

    return createdMember;
  });

  route.patch(
    "/project/:projectId/member/:memberId",
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

      const member = schema.projectMembers.find(request.params.memberId);
      if (!member) {
        return new Response(
          404,
          {},
          {
            errors: "Role mapping id " + request.params.memberId + " not found",
          }
        );
      }
      const oldRole = member.role;

      const attrs = this.normalizedRequestAttrs("project-member-patch");
      const updatedMember = member.update(attrs);

      const ts = Date.now();
      const messageTemplate = {
        containerId: request.params.projectId,
        createdTs: ts,
        updatedTs: ts,
        type: "bb.message.project.member.updaterole",
        status: "DELIVERED",
        creatorId: attrs.updaterId,
        workspaceId: WORKSPACE_ID,
        payload: {
          principalId: member.principalId,
          oldRole,
          newRole: updatedMember.role,
        },
      };
      postMessageToOwnerAndDBA(schema, attrs.updaterId, messageTemplate);

      return updatedMember;
    }
  );

  route.delete(
    "/project/:projectId/member/:memberId",
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

      const member = schema.projectMembers.find(request.params.memberId);
      if (!member) {
        return new Response(
          404,
          {},
          {
            errors:
              "Project role mapping id " +
              request.params.memberId +
              " not found",
          }
        );
      }

      const oldRole = member.role;
      member.destroy();

      const ts = Date.now();
      const messageTemplate = {
        containerId: request.params.projectId,
        createdTs: ts,
        updatedTs: ts,
        type: "bb.message.project.member.revoke",
        status: "DELIVERED",
        creatorId: FAKE_API_CALLER_ID,
        workspaceId: WORKSPACE_ID,
        payload: {
          principalId: member.principalId,
          oldRole,
        },
      };
      postMessageToOwnerAndDBA(schema, FAKE_API_CALLER_ID, messageTemplate);
    }
  );
}
