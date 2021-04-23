import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";
import { postMessageToOwnerAndDBA } from "../utils";

export default function configureAuth(route) {
  route.post("/auth/login", function (schema, request) {
    const loginInfo = this.normalizedRequestAttrs("login-info");
    const user = schema.principals.findBy({
      email: loginInfo.email,
      passwordHash: loginInfo.password,
    });
    if (user) {
      return user;
    }
    return new Response(
      401,
      {},
      { errors: loginInfo.email + " not found or incorrect password" }
    );
  });

  route.post("/auth/signup", function (schema, request) {
    const signupInfo = this.normalizedRequestAttrs("signup-info");
    const user = schema.principals.findBy({ email: signupInfo.email });
    if (user) {
      return new Response(
        409,
        {},
        { errors: signupInfo.email + " already exists" }
      );
    }
    const ts = Date.now();
    const createdUser = schema.principals.create({
      createdTs: ts,
      updatedTs: ts,
      status: "ACTIVE",
      ...signupInfo,
    });

    createdUser.update({
      creatorId: createdUser.id,
      updaterId: createdUser.id,
    });

    const newMember = {
      principalId: createdUser.id,
      email: createdUser.email,
      creatorId: createdUser.id,
      createdTs: ts,
      updaterId: createdUser.id,
      updatedTs: ts,
      role: "DEVELOPER",
      updaterId: createdUser.id,
      workspaceId: WORKSPACE_ID,
    };
    schema.members.create(newMember);

    const messageTemplate = {
      containerId: WORKSPACE_ID,
      createdTs: ts,
      updatedTs: ts,
      type: "bb.msg.member.join",
      status: "DELIVERED",
      creatorId: createdUser.id,
      workspaceId: WORKSPACE_ID,
      payload: {
        principalId: createdUser.id,
        newRole: "DEVELOPER",
      },
    };
    postMessageToOwnerAndDBA(schema, createdUser.id, messageTemplate);

    return createdUser;
  });

  route.post("/auth/activate", function (schema, request) {
    const activateInfo = this.normalizedRequestAttrs("activate-info");
    if (!activateInfo.token) {
      return new Response(400, {}, { errors: "Missing activation token" });
    }

    const user = schema.principals.findBy({ email: activateInfo.email });
    if (user) {
      const ts = Date.now();
      user.update({
        name: activateInfo.name,
        status: "ACTIVE",
        updatedTs: ts,
        passwordHash: activateInfo.password,
      });

      const member = schema.members.findBy({
        principalId: user.id,
        workspaceId: WORKSPACE_ID,
      });

      const messageTemplate = {
        containerId: WORKSPACE_ID,
        createdTs: ts,
        updatedTs: ts,
        type: "bb.msg.member.join",
        status: "DELIVERED",
        creatorId: user.id,
        workspaceId: WORKSPACE_ID,
        payload: {
          principalId: user.id,
          newRole: member.role,
        },
      };
      postMessageToOwnerAndDBA(schema, user.id, messageTemplate);

      return user;
    }

    return new Response(
      400,
      {},
      { errors: activateInfo.email + " is not invited" }
    );
  });
}
