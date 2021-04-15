import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";

export default function configureProject(route) {
  route.get("/project", function (schema, request) {
    const {
      queryParams: { userid: userId, rowstatus: rowStatus },
    } = request;

    return schema.projects.where((project) => {
      if (project.workspaceId != WORKSPACE_ID) {
        return false;
      }

      if (!rowStatus) {
        if (project.rowStatus != "NORMAL") {
          return false;
        }
      } else if (project.rowStatus.toLowerCase() != rowStatus.toLowerCase()) {
        return false;
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
    const newProject = {
      ...this.normalizedRequestAttrs("project-new"),
      workspaceId: WORKSPACE_ID,
    };
    return schema.projects.create(newProject);
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

    return schema.projects.find(request.params.projectId).update(attrs);
  });
}
