import { Response } from "miragejs";
import { postMessageToOwnerAndDBA } from "../utils";
import { WORKSPACE_ID, OWNER_ID } from "./index";
import { DEFAULT_PROJECT_ID, InstanceBuiltinFieldId } from "../../types";

export default function configurInstance(route) {
  route.get("/instance", function (schema, request) {
    const {
      queryParams: { rowstatus: rowStatus },
    } = request;
    return schema.instances.where((instance) => {
      if (instance.workspaceId != WORKSPACE_ID) {
        return false;
      }

      if (!rowStatus) {
        if (instance.rowStatus != "NORMAL") {
          return false;
        }
      } else {
        const rowStatusList = rowStatus.split(",");
        if (
          !rowStatusList.find((item) => {
            return item.toLowerCase() == instance.rowStatus.toLowerCase();
          })
        ) {
          return false;
        }
      }

      return true;
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
    const attrs = this.normalizedRequestAttrs("instance-new");
    const ts = Date.now();
    const createdInstance = schema.instances.create({
      environmentId: attrs.environmentId,
      name: attrs.name,
      externalLink: attrs.externalLink,
      host: attrs.host,
      port: attrs.port,
      rowStatus: "NORMAL",
      creatorId: callerId,
      updaterId: callerId,
      createdTs: ts,
      lastUpdatedTs: ts,
      workspaceId: WORKSPACE_ID,
    });

    const allDatabase = schema.databases.create({
      workspaceId: WORKSPACE_ID,
      projectId: DEFAULT_PROJECT_ID,
      instance: createdInstance,
      name: "*",
    });

    schema.dataSources.create({
      workspaceId: WORKSPACE_ID,
      instance: createdInstance,
      database: allDatabase,
      name: createdInstance.name + " admin data source",
      type: "ADMIN",
      username: attrs.username,
      password: attrs.password,
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

    const dataSource = schema.dataSources.findBy({
      instanceId: instance.id,
      type: "ADMIN",
    });

    if (!dataSource) {
      return new Response(
        404,
        {},
        {
          errors: "Admin connection info for " + instance.name + " not found",
        }
      );
    }

    const attrs = this.normalizedRequestAttrs("instance-patch");
    const instanceAttrs = {
      rowStatus: attrs.rowStatus,
      name: attrs.name,
      externalLink: attrs.externalLink,
      host: attrs.host,
      port: attrs.port,
    };
    const dataSourceAttrs = {
      username: attrs.username,
      password: attrs.password,
    };

    let hasInstanceChange = false;
    let hasDataSourceChange = false;
    const changeList = [];
    let type = "bb.msg.instance.update";

    if (
      instanceAttrs.rowStatus &&
      instanceAttrs.rowStatus != instance.rowStatus
    ) {
      if (instanceAttrs.rowStatus == "ARCHIVED") {
        type = "bb.msg.instance.archive";
      } else if (instanceAttrs.rowStatus == "NORMAL") {
        type = "bb.msg.instance.restore";
      } else if (instanceAttrs.rowStatus == "PENDING_DELETE") {
        type = "bb.msg.instance.delete";
      }
      changeList.push({
        fieldId: InstanceBuiltinFieldId.ROW_STATUS,
        oldValue: instance.rowStatus,
        newValue: instanceAttrs.rowStatus,
      });
      hasInstanceChange = true;
    }

    if (instanceAttrs.name && instanceAttrs.name != instance.name) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.NAME,
        oldValue: instance.name,
        newValue: instanceAttrs.name,
      });
      hasInstanceChange = true;
    }

    if (
      instanceAttrs.externalLink &&
      instanceAttrs.externalLink != instance.externalLink
    ) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.EXTERNAL_LINK,
        oldValue: instance.externalLink,
        newValue: instanceAttrs.externalLink,
      });
      hasInstanceChange = true;
    }

    if (instanceAttrs.host && instanceAttrs.host != instance.host) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.HOST,
        oldValue: instance.host,
        newValue: instanceAttrs.host,
      });
      hasInstanceChange = true;
    }

    if (instanceAttrs.port && instanceAttrs.port != instance.port) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.PORT,
        oldValue: instance.port,
        newValue: instanceAttrs.port,
      });
      hasInstanceChange = true;
    }

    if (
      dataSourceAttrs.username &&
      dataSourceAttrs.username != dataSource.username
    ) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.USERNAME,
        oldValue: dataSource.username,
        newValue: dataSourceAttrs.username,
      });
      hasDataSourceChange = true;
    }

    if (
      dataSourceAttrs.password &&
      dataSourceAttrs.password != dataSource.password
    ) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.PASSWORD,
        oldValue: dataSource.password,
        newValue: dataSourceAttrs.password,
      });
      hasDataSourceChange = true;
    }

    let updatedInstance = instance;

    if (hasInstanceChange) {
      console.log(instanceAttrs);
      updatedInstance = instance.update(instanceAttrs);
    }

    if (hasDataSourceChange) {
      console.log(dataSourceAttrs);
      dataSource.update(dataSourceAttrs);
    }

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
