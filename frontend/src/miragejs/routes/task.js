import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";
import { SYSTEM_BOT_ID } from "../../types";

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

        if (attrs.status == "DONE" || attrs.status == "SKIPPED") {
          const followingTask = schema.tasks.findBy((item) => {
            if (item.workspaceId != WORKSPACE_ID) {
              return false;
            }

            if (item.pipelineId != task.pipelineId) {
              return false;
            }

            return item.id > task.id;
          });

          if (followingTask && followingTask.when == "ON_SUCCESS") {
            const payload = {
              taskId: followingTask.id,
              oldStatus: followingTask.status,
              newStatus: "RUNNING",
            };
            followingTask.update({
              status: "RUNNING",
            });

            schema.activities.create({
              creatorId: SYSTEM_BOT_ID,
              createdTs: ts,
              updaterId: SYSTEM_BOT_ID,
              updatedTs: ts,
              actionType: "bytebase.pipeline.task.status.update",
              containerId: attrs.containerId,
              comment: attrs.comment,
              payload,
              workspaceId: WORKSPACE_ID,
            });
          }
        }

        return updatedTask;
      }

      return task;
    }
  );
}
