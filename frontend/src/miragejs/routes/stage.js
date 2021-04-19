import { Response } from "miragejs";
import isEqual from "lodash-es/isEqual";
import { WORKSPACE_ID } from "./index";
import { TaskBuiltinFieldId } from "../../plugins";
import { UNKNOWN_ID, DEFAULT_PROJECT_ID } from "../../types";

export default function configureStage(route) {
  route.patch(
    "/task/:taskId/stage/:stageId/status",
    function (schema, request) {
      const attrs = this.normalizedRequestAttrs("stage-status-patch");
      const task = schema.tasks.find(request.params.taskId);

      if (!task) {
        return new Response(
          404,
          {},
          { errors: "Task " + request.params.taskId + " not found" }
        );
      }

      const stage = schema.stages.find(request.params.stageId);
      if (!stage) {
        return new Response(
          404,
          {},
          { errors: "Stage " + request.params.stageId + " not found" }
        );
      }

      const ts = Date.now();

      if (attrs.status == "DONE") {
        // We check each steps. Returns error if any of them is not finished.
        const stepList = schema.steps.where({
          taskId: task.id,
          stageId: stage.id,
        }).models;

        for (let j = 0; j < stepList.length; j++) {
          if (step[j].status != "DONE" || step[j].status != "SKIPPED") {
            return new Response(
              404,
              {},
              {
                errors: `Can't resolve task ${task.name}. Step ${step[j].name} in stage ${stage[i].name} is in ${step[j].status} status`,
              }
            );
          }
        }
      }

      const changeList = [];
      const messageList = [];
      const messageTemplate = {
        containerId: task.id,
        creatorId: attrs.updaterId,
        createdTs: ts,
        updaterId: attrs.updaterId,
        lastUpdatedTs: ts,
        status: "DELIVERED",
        workspaceId: WORKSPACE_ID,
      };

      if (attrs.status) {
        if (task.status != attrs.status) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.STAGE_STATUS,
            oldValue: task.status,
            newValue: attrs.status,
          });

          messageList.push({
            ...messageTemplate,
            type: "bb.msg.task.stage.status.update",
            receiverId: task.creatorId,
            payload: {
              taskName: task.name,
              oldStatus: task.status,
              newStatus: attrs.status,
            },
          });

          if (task.assigneeId) {
            messageList.push({
              ...messageTemplate,
              type: "bb.msg.task.stage.status.update",
              receiverId: task.assigneeId,
            });
          }

          for (let subscriberId of task.subscriberIdList) {
            if (
              subscriberId != task.creatorId &&
              subscriberId != task.assigneeId
            ) {
              messageList.push({
                ...messageTemplate,
                type: "bb.msg.task.stage.status.update",
                receiverId: subscriberId,
                payload: {
                  taskName: task.name,
                },
              });
            }
          }
        }
      }

      if (changeList.length) {
        stage.update({ ...attrs, lastUpdatedTs: ts });

        const payload = {
          changeList,
        };

        schema.activities.create({
          creatorId: attrs.updaterId,
          createdTs: ts,
          updaterId: attrs.updaterId,
          lastUpdatedTs: ts,
          actionType: "bytebase.task.stage.status.update",
          containerId: updatedTask.id,
          comment: attrs.comment,
          payload,
          workspaceId: WORKSPACE_ID,
        });

        if (messageList.length > 0) {
          for (const message of messageList) {
            // We only send out message if it's NOT destined to self.
            if (attrs.updaterId != message.receiverId) {
              schema.messages.create(message);
            }
          }
        }

        return updatedTask;
      }

      return task;
    }
  );
}
