import { Response } from "miragejs";
import isEqual from "lodash-es/isEqual";
import { WORKSPACE_ID } from "./index";
import { IssueBuiltinFieldId } from "../../plugins";
import { UNKNOWN_ID, DEFAULT_PROJECT_ID } from "../../types";

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
    const { taskList, ...attrs } = this.normalizedRequestAttrs("issue-new");
    const database = schema.databases.find(taskList[0].databaseId);

    const newIssue = {
      ...attrs,
      createdTs: ts,
      updaterId: attrs.creatorId,
      updatedTs: ts,
      status: "OPEN",
      projectId: database.projectId,
      workspaceId: WORKSPACE_ID,
    };
    const createdIssue = schema.issues.create(newIssue);

    for (const task of taskList) {
      const { stepList, ...taskAttrs } = task;
      const createdTask = schema.tasks.create({
        ...taskAttrs,
        createdTs: ts,
        updaterId: attrs.creatorId,
        updatedTs: ts,
        status: "PENDING",
        issue: createdIssue,
        workspaceId: WORKSPACE_ID,
      });

      for (const step of stepList) {
        schema.steps.create({
          ...step,
          createdTs: ts,
          updaterId: attrs.creatorId,
          updatedTs: ts,
          status: "PENDING",
          issue: createdIssue,
          task: createdTask,
          workspaceId: WORKSPACE_ID,
        });
      }
    }

    schema.activities.create({
      creatorId: attrs.creatorId,
      createdTs: ts,
      updaterId: attrs.updaterId,
      updatedTs: ts,
      actionType: "bytebase.issue.create",
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

        // Send a message to the new assignee
        messageList.push({
          ...messageTemplate,
          type: "bb.msg.issue.assign",
          receiverId: attrs.assigneeId,
          payload: {
            issueName: issue.name,
            oldAssigneeId: issue.assigneeId,
            newAssigneeId: attrs.assigneeId,
          },
        });

        // Send a message to the old assignee
        if (
          issue.assigneeId != UNKNOWN_ID &&
          issue.creatorId != issue.assigneeId
        ) {
          messageList.push({
            ...messageTemplate,
            type: "bb.msg.issue.assign",
            receiverId: issue.assigneeId,
            payload: {
              issueName: issue.name,
              oldAssigneeId: issue.assigneeId,
              newAssigneeId: attrs.assigneeId,
            },
          });
        }

        // Send a message to the creator
        if (issue.creatorId != attrs.assigneeId) {
          messageList.push({
            ...messageTemplate,
            type: "bb.msg.issue.assign",
            receiverId: issue.creatorId,
            payload: {
              issueName: issue.name,
              oldAssigneeId: issue.assigneeId,
              newAssigneeId: attrs.assigneeId,
            },
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

    if (attrs.task !== undefined) {
      const task = issue.taskList.find((item) => item.id == attrs.task.id);
      if (task) {
        changeList.push({
          fieldId: [IssueBuiltinFieldId.TASK, task.id].join("."),
          oldValue: task.status,
          newValue: attrs.task.status,
        });
        task.status = attrs.task.status;
        attrs.taskList = issue.taskList;
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
        actionType: "bytebase.issue.field.update",
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

    if (attrs.status == "DONE") {
      const taskList = schema.tasks.where({ issueId: issue.id }).models;

      // We check each of the task and its steps. Returns error if any of them is not finished.
      for (let i = 0; i < taskList.length; i++) {
        if (task[i].status != "DONE" || task[i].status != "SKIPPED") {
          return new Response(
            404,
            {},
            {
              errors: `Can't resolve issue ${issue.name}. Task ${task[i].name} is in ${task[i].status} status`,
            }
          );
        }

        const stepList = schema.steps.where({
          issueId: issue.id,
          taskId: task[i].id,
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
          fieldId: IssueBuiltinFieldId.STATUS,
          oldValue: issue.status,
          newValue: attrs.status,
        });

        messageList.push({
          ...messageTemplate,
          type: "bb.msg.issue.status.update",
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
            type: "bb.msg.issue.status.update",
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
              type: "bb.msg.issue.status.update",
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
      const updatedIssue = issue.update({ ...attrs, updatedTs: ts });

      const payload = {
        changeList,
      };

      schema.activities.create({
        creatorId: attrs.updaterId,
        createdTs: ts,
        updaterId: attrs.updaterId,
        updatedTs: ts,
        actionType: "bytebase.issue.status.update",
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
}
