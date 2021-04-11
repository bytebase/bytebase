import { Response } from "miragejs";
import { postMessageToOwnerAndDBA } from "../utils";
import { WORKSPACE_ID, OWNER_ID } from "./index";
import { InstanceBuiltinFieldId } from "../../types";

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
    // NOTE, in actual implementation, we need to fetch the user from the auth context.
    const callerId = OWNER_ID;
    const newInstance = {
      ...this.normalizedRequestAttrs("instance-new"),
      workspaceId: WORKSPACE_ID,
    };
    const ts = Date.now();
    const createdInstance = schema.instances.create({
      environmentId: newInstance.environmentId,
      creatorId: callerId,
      updaterId: callerId,
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
      creatorId: callerId,
      workspaceId: WORKSPACE_ID,
      payload: {
        instanceName: createdInstance.name,
      },
    };
    postMessageToOwnerAndDBA(schema, callerId, messageTemplate);

    return createdInstance;
  });

  route.patch("/instance/:instanceId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("instance-patch");
    const instance = schema.instances.find(request.params.instanceId);

    if (!instance) {
      return new Response(
        404,
        {},
        {
          errors: "Instance id " + request.params.instanceId + " not found",
        }
      );
    }

    const changeList = [];
    let type = "bb.msg.instance.update";

    if (attrs.rowStatus && attrs.rowStatus != instance.rowStatus) {
      if (attrs.rowStatus == "ARCHIVED") {
        type = "bb.msg.instance.archive";
      } else if (attrs.rowStatus == "NORMAL") {
        type = "bb.msg.instance.restore";
      } else if (attrs.rowStatus == "PENDING_DELETE") {
        type = "bb.msg.instance.delete";
      }
      changeList.push({
        fieldId: InstanceBuiltinFieldId.ROW_STATUS,
        oldValue: instance.rowStatus,
        newValue: attrs.rowStatus,
      });
    }

    if (attrs.name && attrs.name != instance.name) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.NAME,
        oldValue: instance.name,
        newValue: attrs.name,
      });
    }

    if (attrs.environmentId && attrs.environmentId != instance.environmentId) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.ENVIRONMENT,
        oldValue: instance.environmentId,
        newValue: attrs.environmentId,
      });
    }

    if (attrs.externalLink && attrs.externalLink != instance.externalLink) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.EXTERNAL_LINK,
        oldValue: instance.externalLink,
        newValue: attrs.externalLink,
      });
    }

    if (attrs.host && attrs.host != instance.host) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.HOST,
        oldValue: instance.host,
        newValue: attrs.host,
      });
    }

    if (attrs.port && attrs.port != instance.port) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.PORT,
        oldValue: instance.port,
        newValue: attrs.port,
      });
    }

    const updatedInstance = instance.update(attrs);

    // NOTE, in actual implementation, we need to fetch the user from the auth context.
    const callerId = OWNER_ID;
    const ts = Date.now();
    const messageTemplate = {
      containerId: updatedInstance.id,
      createdTs: ts,
      lastUpdatedTs: ts,
      type,
      status: "DELIVERED",
      creatorId: callerId,
      workspaceId: WORKSPACE_ID,
      payload: {
        instanceName: updatedInstance.name,
        changeList,
      },
    };
    postMessageToOwnerAndDBA(schema, callerId, messageTemplate);

    return updatedInstance;
  });

  route.delete("/instance/:instanceId", function (schema, request) {
    const instance = schema.instances.find(request.params.instanceId);

    if (!instance) {
      return new Response(
        404,
        {},
        {
          errors: "Instance id " + request.params.instanceId + " not found",
        }
      );
    }

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
  });
}
