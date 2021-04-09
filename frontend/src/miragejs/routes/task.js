import { Response } from "miragejs";
import isEqual from "lodash-es/isEqual";
import { WORKSPACE_ID } from "./index";
import { TaskBuiltinFieldId } from "../../plugins";
import { UNKNOWN_ID } from "../../types";

export default function configureTask(route) {
  route.get("/task", function (schema, request) {
    const {
      queryParams: { userid: userId },
    } = request;

    if (userId) {
      return schema.tasks.where((task) => {
        return (
          task.workspaceId == WORKSPACE_ID &&
          (task.creatorId == userId ||
            task.assigneeId == userId ||
            task.subscriberIdList.includes(userId))
        );
      });
    }
    return schema.tasks.none();
  });

  route.get("/task/:id", function (schema, request) {
    const task = schema.tasks.find(request.params.id);
    if (task) {
      return task;
    }
    return new Response(
      404,
      {},
      { errors: "Task " + request.params.id + " not found" }
    );
  });

  route.post("/task", function (schema, request) {
    const ts = Date.now();
    const attrs = this.normalizedRequestAttrs("task");
    const newTask = {
      ...attrs,
      createdTs: ts,
      lastUpdatedTs: ts,
      status: "OPEN",
      workspaceId: WORKSPACE_ID,
    };
    const createdTask = schema.tasks.create(newTask);

    schema.activities.create({
      createdTs: ts,
      lastUpdatedTs: ts,
      actionType: "bytebase.task.create",
      containerId: createdTask.id,
      creatorId: attrs.creatorId,
      comment: "",
      workspaceId: WORKSPACE_ID,
    });

    return createdTask;
  });

  route.patch("/task/:taskId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("task-patch");
    const task = schema.tasks.find(request.params.taskId);
    if (task) {
      const ts = Date.now();
      const changeList = [];
      const messageList = [];
      const messageTemplate = {
        containerId: task.id,
        createdTs: ts,
        lastUpdatedTs: ts,
        status: "DELIVERED",
        creatorId: attrs.updaterId,
        workspaceId: WORKSPACE_ID,
      };

      if (attrs.name) {
        if (task.name != attrs.name) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.NAME,
            oldValue: task.name,
            newValue: attrs.name,
          });
        }
      }

      if (attrs.status) {
        if (task.status != attrs.status) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.STATUS,
            oldValue: task.status,
            newValue: attrs.status,
          });

          messageList.push({
            ...messageTemplate,
            type: "bb.msg.task.updatestatus",
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
              type: "bb.msg.task.updatestatus",
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
                type: "bb.msg.task.updatestatus",
                receiverId: subscriberId,
                payload: {
                  taskName: task.name,
                },
              });
            }
          }
        }
      }

      if (attrs.assigneeId) {
        if (task.assigneeId != attrs.assigneeId) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.ASSIGNEE,
            oldValue: task.assigneeId,
            newValue: attrs.assigneeId,
          });

          // Send a message to the new assignee
          messageList.push({
            ...messageTemplate,
            type: "bb.msg.task.assign",
            receiverId: attrs.assigneeId,
            payload: {
              taskName: task.name,
              oldAssigneeId: task.assigneeId,
              newAssigneeId: attrs.assigneeId,
            },
          });

          // Send a message to the old assignee
          if (
            task.assigneeId != UNKNOWN_ID &&
            task.creatorId != task.assigneeId
          ) {
            messageList.push({
              ...messageTemplate,
              type: "bb.msg.task.assign",
              receiverId: task.assigneeId,
              payload: {
                taskName: task.name,
                oldAssigneeId: task.assigneeId,
                newAssigneeId: attrs.assigneeId,
              },
            });
          }

          // Send a message to the creator
          if (task.creatorId != attrs.assigneeId) {
            messageList.push({
              ...messageTemplate,
              type: "bb.msg.task.assign",
              receiverId: task.creatorId,
              payload: {
                taskName: task.name,
                oldAssigneeId: task.assigneeId,
                newAssigneeId: attrs.assigneeId,
              },
            });
          }
        }
      }

      // Empty string is valid
      if (attrs.description !== undefined) {
        if (task.description != attrs.description) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.DESCRIPTION,
            oldValue: task.description,
            newValue: attrs.description,
          });
        }
      }

      if (attrs.stage !== undefined) {
        const stage = task.stageList.find((item) => item.id == attrs.stage.id);
        if (stage) {
          changeList.push({
            fieldId: [TaskBuiltinFieldId.STAGE, stage.id].join("."),
            oldValue: stage.status,
            newValue: attrs.stage.status,
          });
          stage.status = attrs.stage.status;
          attrs.stageList = task.stageList;
        }
      }

      if (attrs.subscriberIdList !== undefined) {
        if (task.subscriberIdList != attrs.subscriberIdList) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.SUBSCRIBER_LIST,
            oldValue: task.subscriberIdList,
            newValue: attrs.subscriberIdList,
          });
        }
      }

      if (attrs.sql !== undefined) {
        if (task.sql != attrs.sql) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.SQL,
            oldValue: task.sql,
            newValue: attrs.sql,
          });
        }
      }

      if (attrs.rollbackSql !== undefined) {
        if (task.rollbackSql != attrs.rollbackSql) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.ROLLBACK_SQL,
            oldValue: task.rollbackSql,
            newValue: attrs.rollbackSql,
          });
        }
      }

      for (const fieldId in attrs.payload) {
        const oldValue = task.payload[fieldId];
        const newValue = attrs.payload[fieldId];
        if (!isEqual(oldValue, newValue)) {
          changeList.push({
            fieldId: fieldId,
            oldValue: task.payload[fieldId],
            newValue: attrs.payload[fieldId],
          });
        }
      }

      if (changeList.length) {
        const updatedTask = task.update({ ...attrs, lastUpdatedTs: ts });

        const payload = {
          changeList,
        };

        schema.activities.create({
          createdTs: ts,
          lastUpdatedTs: ts,
          actionType: "bytebase.task.field.update",
          containerId: updatedTask.id,
          creatorId: attrs.updaterId,
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
    return new Response(
      404,
      {},
      { errors: "Task " + request.params.id + " not found" }
    );
  });
}
