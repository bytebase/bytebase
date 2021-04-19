import { Response } from "miragejs";
import { postMessageToOwnerAndDBA } from "../utils";
import { WORKSPACE_ID, FAKE_API_CALLER_ID } from "./index";
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
    const attrs = this.normalizedRequestAttrs("instance-new");
    const ts = Date.now();
    const createdInstance = schema.instances.create({
      environmentId: attrs.environmentId,
      name: attrs.name,
      externalLink: attrs.externalLink,
      host: attrs.host,
      port: attrs.port,
      rowStatus: "NORMAL",
      creatorId: attrs.creatorId,
      updaterId: attrs.creatorId,
      createdTs: ts,
      updatedTs: ts,
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
      updatedTs: ts,
      type: "bb.msg.instance.create",
      status: "DELIVERED",
      creatorId: attrs.creatorId,
      workspaceId: WORKSPACE_ID,
      payload: {
        instanceName: createdInstance.name,
      },
    };
    postMessageToOwnerAndDBA(schema, attrs.creatorId, messageTemplate);

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
    const instanceAttrs = {};
    const dataSourceAttrs = {};

    let hasInstanceChange = false;
    let hasDataSourceChange = false;
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
      instanceAttrs.rowStatus = attrs.rowStatus;
      hasInstanceChange = true;
    }

    if (attrs.name && attrs.name != instance.name) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.NAME,
        oldValue: instance.name,
        newValue: attrs.name,
      });
      instanceAttrs.name = attrs.name;
      hasInstanceChange = true;
    }

    if (attrs.externalLink && attrs.externalLink != instance.externalLink) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.EXTERNAL_LINK,
        oldValue: instance.externalLink,
        newValue: attrs.externalLink,
      });
      instanceAttrs.externalLink = attrs.externalLink;
      hasInstanceChange = true;
    }

    if (attrs.host && attrs.host != instance.host) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.HOST,
        oldValue: instance.host,
        newValue: attrs.host,
      });
      instanceAttrs.host = attrs.host;
      hasInstanceChange = true;
    }

    if (attrs.port && attrs.port != instance.port) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.PORT,
        oldValue: instance.port,
        newValue: attrs.port,
      });
      instanceAttrs.port = attrs.port;
      hasInstanceChange = true;
    }

    if (attrs.username && attrs.username != dataSource.username) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.USERNAME,
        oldValue: dataSource.username,
        newValue: attrs.username,
      });
      dataSourceAttrs.username = attrs.username;
      hasDataSourceChange = true;
    }

    if (attrs.password && attrs.password != dataSource.password) {
      changeList.push({
        fieldId: InstanceBuiltinFieldId.PASSWORD,
        oldValue: dataSource.password,
        newValue: attrs.password,
      });
      dataSourceAttrs.password = attrs.password;
      hasDataSourceChange = true;
    }

    let updatedInstance = instance;

    if (hasInstanceChange) {
      updatedInstance = instance.update(instanceAttrs);
    }

    if (hasDataSourceChange) {
      dataSource.update(dataSourceAttrs);
    }

    const ts = Date.now();
    const messageTemplate = {
      containerId: updatedInstance.id,
      createdTs: ts,
      updatedTs: ts,
      type,
      status: "DELIVERED",
      creatorId: attrs.updaterId,
      workspaceId: WORKSPACE_ID,
      payload: {
        instanceName: updatedInstance.name,
        changeList,
      },
    };
    postMessageToOwnerAndDBA(schema, attrs.updaterId, messageTemplate);

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
      updatedTs: ts,
      type: "bb.msg.instance.delete",
      status: "DELIVERED",
      creatorId: FAKE_API_CALLER_ID,
      workspaceId: WORKSPACE_ID,
      payload: {
        instanceName: instance.name,
      },
    };
    postMessageToOwnerAndDBA(schema, FAKE_API_CALLER_ID, messageTemplate);
  });
}
