import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";

export default function configureProject(route) {
  route.get("/project", function (schema, request) {
    const {
      queryParams: { userid: userId },
    } = request;

    return schema.projects.where({
      workspaceId: WORKSPACE_ID,
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
    return schema.projects.find(request.params.projectId).update(attrs);
  });
}
