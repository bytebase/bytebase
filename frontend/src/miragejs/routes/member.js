import { Response } from "miragejs";
import { WORKSPACE_ID, FAKE_API_CALLER_ID } from "./index";
import { postMessageToOwnerAndDBA } from "../utils";

export default function configureMember(route) {
  route.get("/member", function (schema, request) {
    return schema.members.where((member) => {
      return member.workspaceId == WORKSPACE_ID;
    });
  });

  route.post("/member", function (schema, request) {
    const ts = Date.now();
    const attrs = this.normalizedRequestAttrs("member-new");
    const member = schema.members.findBy({
      principalId: attrs.principalId,
      workspaceId: WORKSPACE_ID,
    });
    if (member) {
      return member;
    }

    const newMember = {
      creatorId: attrs.creatorId,
      updaterId: attrs.creatorId,
      createdTs: ts,
      updatedTs: ts,
      role: attrs.role,
      principalId: attrs.principalId,
      workspaceId: WORKSPACE_ID,
    };

    const createdMember = schema.members.create(newMember);

    const user = schema.users.find(attrs.principalId);
    const type =
      user.status == "INVITED"
        ? "bb.msg.member.invite"
        : "bb.msg.member.create";

    const messageTemplate = {
      containerId: WORKSPACE_ID,
      createdTs: ts,
      updatedTs: ts,
      type,
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

  route.patch("/member/:memberId", function (schema, request) {
    const member = schema.members.find(request.params.memberId);
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

    const attrs = this.normalizedRequestAttrs("member");
    const updatedMember = member.update(attrs);

    const ts = Date.now();
    const messageTemplate = {
      containerId: WORKSPACE_ID,
      createdTs: ts,
      updatedTs: ts,
      type: "bb.msg.member.updaterole",
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
  });

  route.delete("/member/:memberId", function (schema, request) {
    const member = schema.members.find(request.params.memberId);

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
    member.destroy();

    const ts = Date.now();
    const messageTemplate = {
      containerId: WORKSPACE_ID,
      createdTs: ts,
      updatedTs: ts,
      type: "bb.msg.member.revoke",
      status: "DELIVERED",
      creatorId: FAKE_API_CALLER_ID,
      workspaceId: WORKSPACE_ID,
      payload: {
        principalId: member.principalId,
        oldRole,
      },
    };
    postMessageToOwnerAndDBA(schema, FAKE_API_CALLER_ID, messageTemplate);
  });
}
