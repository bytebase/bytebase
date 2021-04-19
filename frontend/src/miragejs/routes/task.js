import { Response } from "miragejs";
import isEqual from "lodash-es/isEqual";
import { WORKSPACE_ID } from "./index";
import { IssueBuiltinFieldId } from "../../plugins";
import { UNKNOWN_ID, DEFAULT_PROJECT_ID } from "../../types";

export default function configureTask(route) {
  route.patch(
    "/issue/:issueId/task/:taskId/status",
    function (schema, request) {
      const attrs = this.normalizedRequestAttrs("task-status-patch");
      const issue = schema.issues.find(request.params.issueId);

      if (!issue) {
        return new Response(
          404,
          {},
          { errors: "Issue " + request.params.issueId + " not found" }
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

      if (attrs.status == "DONE") {
        // We check each steps. Returns error if any of them is not finished.
        const stepList = schema.steps.where({
          issueId: issue.id,
          taskId: task.id,
        }).models;

        for (let j = 0; j < stepList.length; j++) {
          if (step[j].status != "DONE" || step[j].status != "SKIPPED") {
            return new Response(
              404,
              {},
              {
                errors: `Can't resolve issue ${issue.name}. Step ${step[j].name} in task ${task[i].name} is in ${step[j].status} status`,
              }
            );
          }
        }
      }

      const changeList = [];
      const messageList = [];
      const messageTemplate = {
        containerId: issue.id,
        creatorId: attrs.updaterId,
        createdTs: ts,
        updaterId: attrs.updaterId,
        updatedTs: ts,
        status: "DELIVERED",
        workspaceId: WORKSPACE_ID,
      };

      if (attrs.status) {
        if (issue.status != attrs.status) {
          changeList.push({
            fieldId: IssueBuiltinFieldId.TASK_STATUS,
            oldValue: issue.status,
            newValue: attrs.status,
          });

          messageList.push({
            ...messageTemplate,
            type: "bb.msg.issue.task.status.update",
            receiverId: issue.creatorId,
            payload: {
              issueName: issue.name,
              oldStatus: issue.status,
              newStatus: attrs.status,
            },
          });

          if (issue.assigneeId) {
            messageList.push({
              ...messageTemplate,
              type: "bb.msg.issue.task.status.update",
              receiverId: issue.assigneeId,
            });
          }

          for (let subscriberId of issue.subscriberIdList) {
            if (
              subscriberId != issue.creatorId &&
              subscriberId != issue.assigneeId
            ) {
              messageList.push({
                ...messageTemplate,
                type: "bb.msg.issue.task.status.update",
                receiverId: subscriberId,
                payload: {
                  issueName: issue.name,
                },
              });
            }
          }
        }
      }

      if (changeList.length) {
        task.update({ ...attrs, updatedTs: ts });

        const payload = {
          changeList,
        };

        schema.activities.create({
          creatorId: attrs.updaterId,
          createdTs: ts,
          updaterId: attrs.updaterId,
          updatedTs: ts,
          actionType: "bytebase.issue.task.status.update",
          containerId: updatedIssue.id,
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

        return updatedIssue;
      }

      return issue;
    }
  );
}
