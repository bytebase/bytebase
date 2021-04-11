import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";
import { postMessageToOwnerAndDBA } from "../utils";

export default function configureAuth(route) {
  route.post("/auth/login", function (schema, request) {
    const loginInfo = this.normalizedRequestAttrs("login-info");
    const user = schema.users.findBy({
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
    const user = schema.users.findBy({ email: signupInfo.email });
    if (user) {
      return new Response(
        409,
        {},
        { errors: signupInfo.email + " already exists" }
      );
    }
    const ts = Date.now();
    const createdUser = schema.users.create({
      createdTs: ts,
      lastUpdatedTs: ts,
      status: "ACTIVE",
      ...signupInfo,
    });

    const newRoleMapping = {
      principalId: createdUser.id,
      email: createdUser.email,
      createdTs: ts,
      lastUpdatedTs: ts,
      role: "DEVELOPER",
      updaterId: createdUser.id,
      workspaceId: WORKSPACE_ID,
    };
    schema.roleMappings.create(newRoleMapping);

    const messageTemplate = {
      containerId: WORKSPACE_ID,
      createdTs: ts,
      lastUpdatedTs: ts,
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

    const user = schema.users.findBy({ email: activateInfo.email });
    if (user) {
      const ts = Date.now();
      user.update({
        name: activateInfo.name,
        status: "ACTIVE",
        lastUpdatedTs: ts,
        passwordHash: activateInfo.password,
      });

      const roleMapping = schema.roleMappings.findBy({
        principalId: user.id,
        workspaceId: WORKSPACE_ID,
      });

      const messageTemplate = {
        containerId: WORKSPACE_ID,
        createdTs: ts,
        lastUpdatedTs: ts,
        type: "bb.msg.member.join",
        status: "DELIVERED",
        creatorId: user.id,
        workspaceId: WORKSPACE_ID,
        payload: {
          principalId: user.id,
          newRole: roleMapping.role,
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
