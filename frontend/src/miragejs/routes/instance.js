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

    const messageList = [];
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

    const allOwnerAndDBAs = schema.roleMappings.where((roleMapping) => {
      return (
        roleMapping.workspaceId == WORKSPACE_ID &&
        (roleMapping.role == "OWNER" || roleMapping.role == "DBA")
      );
    }).models;

    allOwnerAndDBAs.forEach((roleMapping) => {
      messageList.push({
        ...messageTemplate,
        receiverId: roleMapping.principalId,
      });
    });

    for (const message of messageList) {
      // We only send out message if it's NOT destined to self.
      if (newInstance.creatorId != message.receiverId) {
        schema.messages.create(message);
      }
    }

    return createdInstance;
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
