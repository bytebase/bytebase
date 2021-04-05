import { WORKSPACE_ID } from "./index";

export default function configureEnvironment(route) {
  route.get("/environment", function (schema, request) {
    return schema.environments
      .where((environment) => {
        return environment.workspaceId == WORKSPACE_ID;
      })
      .sort((a, b) => a.order - b.order);
  });

  route.post("/environment", function (schema, request) {
    let order = 0;
    const list = schema.environments.where((environment) => {
      return environment.workspaceId == WORKSPACE_ID;
    });
    if (list.length > 0) {
      order = list.sort((a, b) => b.order - a.order).models[0].order + 1;
    }
    const newEnvironment = {
      ...this.normalizedRequestAttrs("environment"),
      workspaceId: WORKSPACE_ID,
      order,
    };
    return schema.environments.create(newEnvironment);
  });

  route.patch("/environment/batch", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("batch-update");
    for (let i = 0; i < attrs.idList.length; i++) {
      const env = schema.environments.find(attrs.idList[i]);
      if (env) {
        const batch = {};
        for (let j = 0; j < attrs.fieldMaskList.length; j++) {
          batch[attrs.fieldMaskList[j]] = attrs.rowValueList[i][j];
        }
        env.update(batch);
      }
    }
    return schema.environments
      .where((environment) => {
        return environment.workspaceId == WORKSPACE_ID;
      })
      .sort((a, b) => a.order - b.order);
  });

  route.patch("/environment/:environmentId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("environment");
    return schema.environments.find(request.params.environmentId).update(attrs);
  });

  route.delete("/environment/:environmentId", function (schema, request) {
    return schema.environments.find(request.params.environmentId).destroy();
  });
}
