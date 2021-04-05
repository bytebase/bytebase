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
