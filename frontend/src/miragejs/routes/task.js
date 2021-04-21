import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";
import { IssueBuiltinFieldId } from "../../plugins";

export default function configureTask(route) {
  route.patch(
    "/pipeline/:pipelineId/task/:taskId/status",
    function (schema, request) {
      const attrs = this.normalizedRequestAttrs("task-status-patch");
      const pipeline = schema.pipelines.find(request.params.pipelineId);

      if (!pipeline) {
        return new Response(
          404,
          {},
          { errors: "Pipeline " + request.params.pipelineId + " not found" }
        );
      }

      const task = schema.tasks.find(request.params.taskId);
      if (!task) {
        return new Response(
          404,
          {},
          { errors: "Task " + request.params.taskId + " not found" }
        );
      }

      const ts = Date.now();

      if (attrs.status && task.status != attrs.status) {
        const payload = {
          taskId: task.id,
          oldStatus: task.status,
          newStatus: attrs.status,
        };

        const updatedTask = task.update({ ...attrs, updatedTs: ts });

        schema.activities.create({
          creatorId: attrs.updaterId,
          createdTs: ts,
          updaterId: attrs.updaterId,
          updatedTs: ts,
          actionType: "bytebase.pipeline.task.status.update",
          containerId: attrs.containerId,
          comment: attrs.comment,
          payload,
          workspaceId: WORKSPACE_ID,
        });

        return updatedTask;
      }

      return task;
    }
  );
}
