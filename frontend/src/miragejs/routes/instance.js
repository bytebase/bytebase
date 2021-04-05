import { WORKSPACE_ID } from "./index";

export default function configurInstance(route) {
  route.get("/instance", function (schema, request) {
    return schema.instances.where((instance) => {
      return instance.workspaceId == WORKSPACE_ID;
    });
  });

  route.get("/instance/:id", function (schema, request) {
    const instance = schema.instances.find(request.params.id);
    if (instance) {
      return instance;
    }
    return new Response(
      404,
      {},
      { errors: "Instance " + request.params.id + " not found" }
    );
  });

  route.post("/instance", function (schema, request) {
    const newInstance = {
      ...this.normalizedRequestAttrs("instance"),
      workspaceId: WORKSPACE_ID,
    };
    return schema.instances.create(newInstance);
  });

  route.patch("/instance/:instanceId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("instance");
    return schema.instances.find(request.params.instanceId).update(attrs);
  });

  route.delete("/instance/:instanceId", function (schema, request) {
    // Delete data source and database before instance itself.
    // Otherwise, the instanceId will be set to null.
    const dataSourceList = schema.dataSources.where({
      instanceId: request.params.instanceId,
    });
    dataSourceList.models.forEach((item) => item.destroy());

    const databaseList = schema.databases.where({
      instanceId: request.params.instanceId,
    });
    databaseList.models.forEach((item) => item.destroy());

    return schema.instances.find(request.params.instanceId).destroy();
  });
}
