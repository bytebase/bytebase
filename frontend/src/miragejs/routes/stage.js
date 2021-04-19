import { Response } from "miragejs";
import isEqual from "lodash-es/isEqual";
import { WORKSPACE_ID } from "./index";
import { IssueBuiltinFieldId } from "../../plugins";
import { UNKNOWN_ID, DEFAULT_PROJECT_ID } from "../../types";

export default function configureStage(route) {
  route.patch(
    "/issue/:issueId/stage/:stageId/status",
    function (schema, request) {
      const attrs = this.normalizedRequestAttrs("stage-status-patch");
      const issue = schema.issues.find(request.params.issueId);

      if (!issue) {
        return new Response(
          404,
          {},
          { errors: "Issue " + request.params.issueId + " not found" }
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
          issueId: issue.id,
          stageId: stage.id,
        }).models;

        for (let j = 0; j < stepList.length; j++) {
          if (step[j].status != "DONE" || step[j].status != "SKIPPED") {
            return new Response(
              404,
              {},
              {
                errors: `Can't resolve issue ${issue.name}. Step ${step[j].name} in stage ${stage[i].name} is in ${step[j].status} status`,
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
            fieldId: IssueBuiltinFieldId.STAGE_STATUS,
            oldValue: issue.status,
            newValue: attrs.status,
          });

          messageList.push({
            ...messageTemplate,
            type: "bb.msg.issue.stage.status.update",
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
              type: "bb.msg.issue.stage.status.update",
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
                type: "bb.msg.issue.stage.status.update",
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
        stage.update({ ...attrs, updatedTs: ts });

        const payload = {
          changeList,
        };

        schema.activities.create({
          creatorId: attrs.updaterId,
          createdTs: ts,
          updaterId: attrs.updaterId,
          updatedTs: ts,
          actionType: "bytebase.issue.stage.status.update",
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
