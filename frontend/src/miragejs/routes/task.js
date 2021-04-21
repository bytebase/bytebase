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

      const changeList = [];

      if (attrs.status && task.status != attrs.status) {
        changeList.push({
          fieldId: IssueBuiltinFieldId.TASK_STATUS,
          oldValue: task.status,
          newValue: attrs.status,
        });

        const updatedTask = task.update({ ...attrs, updatedTs: ts });

        const payload = {
          changeList,
        };

        schema.activities.create({
          creatorId: attrs.updaterId,
          createdTs: ts,
          updaterId: attrs.updaterId,
          updatedTs: ts,
          actionType: "bytebase.issue.stage.task.status.update",
          containerId: updatedTask.id,
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
