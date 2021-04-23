import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";

export default function configureActivity(route) {
  route.get("/activity", function (schema, request) {
    const {
      queryParams: { container: containerId },
    } = request;
    return schema.activities.where((activity) => {
      if (activity.workspaceId != WORKSPACE_ID) {
        return false;
      }

      if (containerId && containerId != activity.containerId) {
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
      updaterId: attrs.creatorId,
      updatedTs: ts,
      actionType: "bb.issue.comment.create",
      workspaceId: WORKSPACE_ID,
    };
    const createdActivity = schema.activities.create(newActivity);

    const issue = schema.issues.find(attrs.containerId);

    if (issue) {
      const messageList = [];
      const messageTemplate = {
        containerId: attrs.containerId,
        createdTs: ts,
        updatedTs: ts,
        type: "bb.message.issue.comment",
        status: "DELIVERED",
        description: attrs.comment,
        creatorId: attrs.creatorId,
        workspaceId: WORKSPACE_ID,
        payload: {
          issueName: issue.name,
          commentId: createdActivity.id,
        },
      };

      messageList.push({
        ...messageTemplate,
        receiverId: issue.creatorId,
      });

      if (issue.assigneeId) {
        messageList.push({
          ...messageTemplate,
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
      return activity.update({ ...attrs, updatedTs: Date.now() });
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
