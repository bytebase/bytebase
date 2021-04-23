import { Response } from "miragejs";
import isEqual from "lodash-es/isEqual";
import { WORKSPACE_ID } from "./index";
import { IssueBuiltinFieldId } from "../../plugins";
import { UNKNOWN_ID, DEFAULT_PROJECT_ID, EMPTY_ID } from "../../types";
import { postIssueMessageToReceiver } from "../utils";

export default function configureIssue(route) {
  route.get("/issue", function (schema, request) {
    const {
      queryParams: { user: userId, project: projectId },
    } = request;

    if (userId || projectId) {
      return schema.issues.where((issue) => {
        return (
          issue.workspaceId == WORKSPACE_ID &&
          (!userId ||
            issue.creatorId == userId ||
            issue.assigneeId == userId ||
            issue.subscriberIdList.includes(userId)) &&
          (!projectId || issue.projectId == projectId)
        );
      });
    }
    return schema.issues.all();
  });

  route.get("/issue/:id", function (schema, request) {
    const issue = schema.issues.find(request.params.id);
    if (issue) {
      return issue;
    }
    return new Response(
      404,
      {},
      { errors: "Issue " + request.params.id + " not found" }
    );
  });

  route.post("/issue", function (schema, request) {
    const ts = Date.now();
    const { pipeline, ...attrs } = this.normalizedRequestAttrs("issue-new");

    let createdPipeline;
    // Create pipeline if exists
    if (pipeline) {
      const newPipeline = {
        createdTs: ts,
        updaterId: attrs.creatorId,
        updatedTs: ts,
        name: pipeline.name,
        status: "OPEN",
        workspaceId: WORKSPACE_ID,
      };

      createdPipeline = schema.pipelines.create(newPipeline);

      for (const stage of pipeline.stageList) {
        const { taskList, databaseId, environmentId, ...stageAttrs } = stage;

        const createdStage = schema.stages.create({
          ...stageAttrs,
          createdTs: ts,
          updaterId: attrs.creatorId,
          updatedTs: ts,
          environmentId,
          databaseId: databaseId != EMPTY_ID ? databaseId : null,
          status: "PENDING",
          pipeline: createdPipeline,
          workspaceId: WORKSPACE_ID,
        });

        for (const task of taskList) {
          schema.tasks.create({
            ...task,
            createdTs: ts,
            updaterId: attrs.creatorId,
            updatedTs: ts,
            status: "PENDING",
            pipeline: createdPipeline,
            stage: createdStage,
            workspaceId: WORKSPACE_ID,
          });
        }
      }
    }

    const newIssue = {
      ...attrs,
      createdTs: ts,
      updaterId: attrs.creatorId,
      updatedTs: ts,
      status: "OPEN",
      subscriberIdList: [],
      pipeline: createdPipeline,
      workspaceId: WORKSPACE_ID,
    };

    const createdIssue = schema.issues.create(newIssue);
    schema.activities.create({
      creatorId: attrs.creatorId,
      createdTs: ts,
      updaterId: attrs.updaterId,
      updatedTs: ts,
      actionType: "bb.issue.create",
      containerId: createdIssue.id,
      comment: "",
      workspaceId: WORKSPACE_ID,
    });

    return createdIssue;
  });

  route.patch("/issue/:issueId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("issue-patch");
    const issue = schema.issues.find(request.params.issueId);

    if (!issue) {
      return new Response(
        404,
        {},
        { errors: "Issue " + request.params.id + " not found" }
      );
    }

    const ts = Date.now();
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

    if (attrs.assigneeId) {
      if (issue.assigneeId != attrs.assigneeId) {
        changeList.push({
          fieldId: IssueBuiltinFieldId.ASSIGNEE,
          oldValue: issue.assigneeId,
          newValue: attrs.assigneeId,
        });

        messageTemplate.type = "bb.message.issue.assign";
        messageTemplate.payload = {
          issueName: issue.name,
          oldAssigneeId: issue.assigneeId,
          newAssigneeId: attrs.assigneeId,
        };

        // Send a message to the new assignee
        messageList.push({
          ...messageTemplate,
          receiverId: attrs.assigneeId,
        });

        // Send a message to the old assignee
        if (
          issue.assigneeId != UNKNOWN_ID &&
          issue.creatorId != issue.assigneeId
        ) {
          messageList.push({
            ...messageTemplate,
            receiverId: issue.assigneeId,
          });
        }

        // Send a message to the creator
        if (issue.creatorId != attrs.assigneeId) {
          messageList.push({
            ...messageTemplate,
            receiverId: issue.creatorId,
          });
        }
      }
    }

    // Empty string is valid
    if (attrs.description !== undefined) {
      if (issue.description != attrs.description) {
        changeList.push({
          fieldId: IssueBuiltinFieldId.DESCRIPTION,
          oldValue: issue.description,
          newValue: attrs.description,
        });
      }
    }

    if (attrs.stage !== undefined) {
      const stage = issue.stageList.find((item) => item.id == attrs.stage.id);
      if (stage) {
        changeList.push({
          fieldId: [IssueBuiltinFieldId.STAGE, stage.id].join("."),
          oldValue: stage.status,
          newValue: attrs.stage.status,
        });
        stage.status = attrs.stage.status;
        attrs.stageList = issue.stageList;
      }
    }

    if (attrs.subscriberIdList !== undefined) {
      if (issue.subscriberIdList != attrs.subscriberIdList) {
        changeList.push({
          fieldId: IssueBuiltinFieldId.SUBSCRIBER_LIST,
          oldValue: issue.subscriberIdList,
          newValue: attrs.subscriberIdList,
        });
      }
    }

    if (attrs.sql !== undefined) {
      if (issue.sql != attrs.sql) {
        changeList.push({
          fieldId: IssueBuiltinFieldId.SQL,
          oldValue: issue.sql,
          newValue: attrs.sql,
        });
      }
    }

    if (attrs.rollbackSql !== undefined) {
      if (issue.rollbackSql != attrs.rollbackSql) {
        changeList.push({
          fieldId: IssueBuiltinFieldId.ROLLBACK_SQL,
          oldValue: issue.rollbackSql,
          newValue: attrs.rollbackSql,
        });
      }
    }

    for (const fieldId in attrs.payload) {
      const oldValue = issue.payload[fieldId];
      const newValue = attrs.payload[fieldId];
      if (!isEqual(oldValue, newValue)) {
        changeList.push({
          fieldId: fieldId,
          oldValue: issue.payload[fieldId],
          newValue: attrs.payload[fieldId],
        });
      }
    }

    if (changeList.length) {
      const updatedIssue = issue.update({ ...attrs, updatedTs: ts });

      const payload = {
        changeList,
      };

      schema.activities.create({
        creatorId: attrs.updaterId,
        createdTs: ts,
        updaterId: attrs.updaterId,
        updatedTs: ts,
        actionType: "bb.issue.field.update",
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
  });

  route.patch("/issue/:issueId/status", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("issue-status-patch");
    const issue = schema.issues.find(request.params.issueId);

    if (!issue) {
      return new Response(
        404,
        {},
        { errors: "Issue " + request.params.id + " not found" }
      );
    }

    const ts = Date.now();

    if (issue.pipelineId) {
      const pipeline = schema.pipelines.find(issue.pipelineId);
      // Pipeline and issue status is 1-to-1 mapping, so we just change the pipeline status accordingly.
      pipeline.update({
        status: attrs.status,
      });

      const stageList = schema.stages.where({ pipelineId: pipeline.id }).models;
      if (attrs.status == "DONE") {
        // Returns error if any of the tasks is not in the end status.
        for (let i = 0; i < stageList.length; i++) {
          const taskList = schema.tasks.where({
            issueId: issue.id,
            stageId: stageList[i].id,
          }).models;

          for (let j = 0; j < taskList.length; j++) {
            if (
              taskList[j].status != "DONE" &&
              taskList[j].status != "CANCELED" &&
              taskList[j].status != "SKIPPED"
            ) {
              return new Response(
                404,
                {},
                {
                  errors: `Can't resolve issue ${issue.name}. Task ${taskList[j].name} in stage ${stageList[i].name} is in ${taskList[j].status} status`,
                }
              );
            }
          }
        }

        pipeline.update({ status: "DONE" });
      }

      // If issue is canceled, we find the current running stages and tasks, mark each of them CANCELED.
      // We keep PENDING stages and tasks as is since the issue maybe reopened later, and it's better to
      // keep them in the state before it was canceled.
      if (attrs.status == "CANCELED") {
        pipeline.update({ status: "CAMCELED" });

        for (let i = 0; i < stageList.length; i++) {
          if (stageList[i].status == "RUNNING") {
            schema.stages.find(stageList[i].id).update({
              status: "CANCELED",
            });

            const taskList = schema.tasks.where({
              issueId: issue.id,
              stageId: stageList[i].id,
            }).models;

            for (let j = 0; j < taskList.length; j++) {
              if (taskList[j].status == "RUNNING") {
                schema.tasks.find(taskList[j].id).update({
                  status: "CANCELED",
                });
              }
            }
          }
        }
      }

      // If issue is opened, we just move the pipeline to the PENDING status.
      // We keep stages and tasks status as is since even those status are canceled,
      // we don't known whether it's canceled because of the issue is previously
      // canceled, or it's canceled for a different reason. And it's always safer
      // for user to explicitly resume the execution.
      if (attrs.status == "OPEN") {
        pipeline.update({ status: "PENDING" });
      }
    }

    const changeList = [];
    const messageTemplate = {
      containerId: issue.id,
      creatorId: attrs.updaterId,
      createdTs: ts,
      updaterId: attrs.updaterId,
      updatedTs: ts,
      status: "DELIVERED",
      workspaceId: WORKSPACE_ID,
    };

    if (attrs.status && issue.status != attrs.status) {
      changeList.push({
        fieldId: IssueBuiltinFieldId.STATUS,
        oldValue: issue.status,
        newValue: attrs.status,
      });

      messageTemplate.type = "bb.message.issue.status.update";
      messageTemplate.payload = {
        issueName: issue.name,
        oldStatus: issue.status,
        newStatus: attrs.status,
      };
    }

    if (changeList.length) {
      const updatedIssue = issue.update({ ...attrs, updatedTs: ts });

      const payload = {
        changeList,
      };

      schema.activities.create({
        creatorId: attrs.updaterId,
        createdTs: ts,
        updaterId: attrs.updaterId,
        updatedTs: ts,
        actionType: "bb.issue.status.update",
        containerId: updatedIssue.id,
        comment: attrs.comment,
        payload,
        workspaceId: WORKSPACE_ID,
      });

      postIssueMessageToReceiver(
        schema,
        updatedIssue,
        attrs.updaterId,
        messageTemplate
      );

      return updatedIssue;
    }

    return issue;
  });
}
