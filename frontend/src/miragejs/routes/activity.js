import { WORKSPACE_ID } from "./index";

export default function configureActivity(route) {
  route.get("/activity", function (schema, request) {
    const {
      queryParams: { containerid: containerId, type },
    } = request;
    return schema.activities.where((activity) => {
      if (activity.workspaceId != WORKSPACE_ID) {
        return false;
      }

      if (containerId && containerId != activity.containerId) {
        return false;
      }

      if (type && !activity.actionType.startsWith(type)) {
        return false;
      }

      return true;
    });
  });

  route.post("/activity", function (schema, request) {
    const ts = Date.now();
    const attrs = this.normalizedRequestAttrs("activity");
    const newActivity = {
      ...attrs,
      createdTs: ts,
      lastUpdatedTs: ts,
      actionType: "bytebase.task.comment.create",
      workspaceId: WORKSPACE_ID,
    };
    const createdActivity = schema.activities.create(newActivity);

    const task = schema.tasks.find(attrs.containerId);

    if (task) {
      const messageList = [];
      const messageTemplate = {
        containerId: attrs.containerId,
        createdTs: ts,
        lastUpdatedTs: ts,
        type: "bb.msg.task.comment",
        status: "DELIVERED",
        description: attrs.comment,
        creatorId: attrs.creatorId,
        workspaceId: WORKSPACE_ID,
        payload: {
          taskName: task.name,
        },
      };

      messageList.push({
        ...messageTemplate,
        receiverId: task.creatorId,
      });

      if (task.assigneeId) {
        messageList.push({
          ...messageTemplate,
          receiverId: task.assigneeId,
        });
      }

      for (let subscriberId of task.subscriberIdList) {
        if (subscriberId != task.creatorId && subscriberId != task.assigneeId) {
          messageList.push({
            ...messageTemplate,
            receiverId: subscriberId,
          });
        }
      }

      if (messageList.length > 0) {
        for (const message of messageList) {
          // We only send out message if it's NOT destined to self.
          if (attrs.creatorId != message.receiverId) {
            schema.messages.create(message);
          }
        }
      }
    }

    return createdActivity;
  });

  route.patch("/activity/:activityId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("activity-patch");
    const activity = schema.activities.find(request.params.activityId);
    if (activity) {
      return activity.update({ ...attrs, lastUpdatedTs: Date.now() });
    }
    return new Response(
      404,
      {},
      { errors: "Activity " + request.params.activityId + " not found" }
    );
  });

  route.delete("/activity/:activityId", function (schema, request) {
    return schema.activities.find(request.params.activityId).destroy();
  });
}
