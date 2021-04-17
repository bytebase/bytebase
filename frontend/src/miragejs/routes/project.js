import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";
import { DEFAULT_PROJECT_ID } from "../../types";

export default function configureProject(route) {
  route.get("/project", function (schema, request) {
    const {
      queryParams: { user: userId, rowstatus: rowStatus },
    } = request;

    return schema.projects.where((project) => {
      if (project.id == DEFAULT_PROJECT_ID) {
        return false;
      }

      if (project.workspaceId != WORKSPACE_ID) {
        return false;
      }

      if (!rowStatus) {
        if (project.rowStatus != "NORMAL") {
          return false;
        }
      } else {
        const rowStatusList = rowStatus.split(",");
        if (
          !rowStatusList.find((item) => {
            return item.toLowerCase() == project.rowStatus.toLowerCase();
          })
        ) {
          return false;
        }
      }

      if (userId) {
        return (
          schema.projectMembers.findBy({
            projectId: project.id,
            principalId: userId,
          }) != undefined
        );
      }

      return true;
    });
  });

  route.get("/project/:id", function (schema, request) {
    const project = schema.projects.find(request.params.id);
    if (!project) {
      return new Response(
        404,
        {},
        { errors: "Project id " + request.params.id + " not found" }
      );
    }

    return project;
  });

  route.post("/project", function (schema, request) {
    const ts = Date.now();
    const attrs = this.normalizedRequestAttrs("project-new");
    const newProject = {
      ...attrs,
      creatorId: attrs.creatorId,
      createdTs: ts,
      updaterId: attrs.creatorId,
      lastUpdatedTs: ts,
      workspaceId: WORKSPACE_ID,
    };
    const project = schema.projects.create(newProject);

    schema.projectMembers.create({
      creatorId: attrs.creatorId,
      updaterId: attrs.creatorId,
      createdTs: ts,
      lastUpdatedTs: ts,
      projectId: project.id,
      role: "OWNER",
      principalId: attrs.creatorId,
    });

    return project;
  });

  route.patch("/project/:projectId", function (schema, request) {
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

    if (project.id == DEFAULT_PROJECT_ID) {
      return new Response(400, {}, { errors: "Can't update default project" });
    }

    const attrs = this.normalizedRequestAttrs("project-patch");
    if (attrs.key) {
      if (schema.projects.findBy({ key: attrs.key })) {
        return new Response(
          409,
          {},
          {
            errors: `Project key ${attrs.key} already exists, please choose a different key.`,
          }
        );
      }
    }

    return schema.projects.find(request.params.projectId).update({
      ...attrs,
      updaterId: attrs.updaterId,
      lastUpdatedTs: Date.now(),
    });
  });
}
