import { postMessageToOwnerAndDBA } from "../utils";
import { WORKSPACE_ID, OWNER_ID } from "./index";

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
      ...this.normalizedRequestAttrs("instance-new"),
      workspaceId: WORKSPACE_ID,
    };
    const ts = Date.now();
    const createdInstance = schema.instances.create({
      environmentId: newInstance.environmentId,
      creatorId: newInstance.creatorId,
      updaterId: newInstance.creatorId,
      createdTs: ts,
      lastUpdatedTs: ts,
      name: newInstance.name,
      externalLink: newInstance.externalLink,
      host: newInstance.host,
      port: newInstance.port,
    });

    const messageTemplate = {
      containerId: createdInstance.id,
      createdTs: ts,
      lastUpdatedTs: ts,
      type: "bb.msg.instance.create",
      status: "DELIVERED",
      creatorId: newInstance.creatorId,
      workspaceId: WORKSPACE_ID,
      payload: {
        instanceName: createdInstance.name,
      },
    };
    postMessageToOwnerAndDBA(schema, newInstance.creatorId, messageTemplate);

    return createdInstance;
  });

  route.patch("/instance/:instanceId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("instance");
    return schema.instances.find(request.params.instanceId).update(attrs);
  });

  route.delete("/instance/:instanceId", function (schema, request) {
    // NOTE, in actual implementation, we need to fetch the user from the auth context.
    const callerId = OWNER_ID;
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

    const instance = schema.instances.find(request.params.instanceId);

    if (instance) {
      instance.destroy();

      const ts = Date.now();
      const messageTemplate = {
        containerId: instance.id,
        createdTs: ts,
        lastUpdatedTs: ts,
        type: "bb.msg.instance.delete",
        status: "DELIVERED",
        creatorId: callerId,
        workspaceId: WORKSPACE_ID,
        payload: {
          instanceName: instance.name,
        },
      };
      postMessageToOwnerAndDBA(schema, callerId, messageTemplate);
    }
  });
}
