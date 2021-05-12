import { Response } from "miragejs";
import { postMessageToOwnerAndDBA } from "../utils";
import { WORKSPACE_ID } from "./index";
import { EnvironmentBuiltinFieldId } from "../../types";

export default function configureEnvironment(route) {
  route.get("/environment", function (schema, request) {
    const {
      queryParams: { rowstatus: rowStatus },
    } = request;
    return schema.environments.where((environment) => {
      if (environment.workspaceId != WORKSPACE_ID) {
        return false;
      }

      if (!rowStatus) {
        if (environment.rowStatus != "NORMAL") {
          return false;
        }
      } else {
        const rowStatusList = rowStatus.split(",");
        if (
          !rowStatusList.find((item) => {
            return item.toLowerCase() == environment.rowStatus.toLowerCase();
          })
        ) {
          return false;
        }
      }

      return true;
    });
  });

  route.post("/environment", function (schema, request) {
    let order = 0;
    const list = schema.environments.where((environment) => {
      return environment.workspaceId == WORKSPACE_ID;
    });
    if (list.length > 0) {
      order = list.sort((a, b) => b.order - a.order).models[0].order + 1;
    }

    const attrs = this.normalizedRequestAttrs("environment-create");
    const ts = Date.now();
    const newEnvironment = {
      ...attrs,
      creatorId: attrs.creatorId,
      createdTs: ts,
      updaterId: attrs.creatorId,
      updatedTs: ts,
      workspaceId: WORKSPACE_ID,
      rowStatus: "NORMAL",
      order,
    };
    const createdEnvironment = schema.environments.create(newEnvironment);

    const messageTemplate = {
      containerId: createdEnvironment.id,
      createdTs: ts,
      updatedTs: ts,
      type: "bb.message.environment.create",
      status: "DELIVERED",
      creatorId: attrs.creatorId,
      workspaceId: WORKSPACE_ID,
      payload: {
        environmentName: createdEnvironment.name,
      },
    };
    postMessageToOwnerAndDBA(schema, attrs.creatorId, messageTemplate);

    return createdEnvironment;
  });

  route.patch("/environment/reorder", function (schema, request) {
    const list = JSON.parse(request.requestBody).data;
    let updaterId;
    let updated = false;
    for (let i = 0; i < list.length; i++) {
      const env = schema.environments.find(list[i].id);
      if (env) {
        updaterId = list[i].attributes.updaterId;
        const oneUpdate = {
          updaterId: list[i].attributes.updaterId,
          order: list[i].attributes.order,
        };
        env.update(oneUpdate);
        updated = true;
      }
    }

    if (updated) {
      const ts = Date.now();
      const messageTemplate = {
        containerId: WORKSPACE_ID,
        createdTs: ts,
        updatedTs: ts,
        type: "bb.message.environment.reorder",
        status: "DELIVERED",
        creatorId: updaterId,
        workspaceId: WORKSPACE_ID,
      };
      postMessageToOwnerAndDBA(schema, updaterId, messageTemplate);
    }

    return schema.environments.where((environment) => {
      return environment.workspaceId == WORKSPACE_ID;
    });
  });

  route.patch("/environment/:environmentId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("environment-patch");
    const environment = schema.environments.find(request.params.environmentId);

    if (!environment) {
      return new Response(
        404,
        {},
        {
          errors:
            "Environment id " + request.params.environmentId + " not found",
        }
      );
    }

    const changeList = [];
    let type = "bb.message.environment.update";

    if (attrs.rowStatus && attrs.rowStatus != environment.rowStatus) {
      if (attrs.rowStatus == "ARCHIVED") {
        type = "bb.message.environment.archive";
      } else if (attrs.rowStatus == "NORMAL") {
        type = "bb.message.environment.restore";
      } else if (attrs.rowStatus == "PENDING_DELETE") {
        type = "bb.message.environment.delete";
      }
      changeList.push({
        fieldId: EnvironmentBuiltinFieldId.ROW_STATUS,
        oldValue: environment.rowStatus,
        newValue: attrs.rowStatus,
      });
    }

    if (attrs.name && attrs.name != environment.name) {
      changeList.push({
        fieldId: EnvironmentBuiltinFieldId.NAME,
        oldValue: environment.name,
        newValue: attrs.name,
      });
    }

    const updatedEnvironment = environment.update(attrs);

    const ts = Date.now();
    const messageTemplate = {
      containerId: environment.id,
      createdTs: ts,
      updatedTs: ts,
      type,
      status: "DELIVERED",
      creatorId: attrs.updaterId,
      workspaceId: WORKSPACE_ID,
      payload: {
        environmentName: updatedEnvironment.name,
        changeList,
      },
    };
    postMessageToOwnerAndDBA(schema, attrs.updaterId, messageTemplate);

    return updatedEnvironment;
  });

  route.delete("/environment/:environmentId", function (schema, request) {
    const environment = schema.environments.find(request.params.environmentId);

    if (!environment) {
      return new Response(
        404,
        {},
        {
          errors:
            "Environment id " + request.params.environmentId + " not found",
        }
      );
    }

    const instanceCount = schema.instances.where({
      environmentId: request.params.environmentId,
    }).models.length;
    if (instanceCount > 0) {
      return new Response(
        400,
        {},
        {
          errors: `Found ${instanceCount} instance(s) in environment ${environment.name}. Environment can only be deleted if no instance resides under it.`,
        }
      );
    }

    environment.destroy();
  });
}
