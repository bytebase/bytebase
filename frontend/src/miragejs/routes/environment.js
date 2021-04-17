import { Response } from "miragejs";
import { postMessageToOwnerAndDBA } from "../utils";
import { WORKSPACE_ID, FAKE_API_CALLER_ID } from "./index";
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
    const newEnvironment = {
      ...this.normalizedRequestAttrs("environment-new"),
      workspaceId: WORKSPACE_ID,
      rowStatus: "NORMAL",
      order,
    };
    const createdEnvironment = schema.environments.create(newEnvironment);

    const ts = Date.now();
    const messageTemplate = {
      containerId: createdEnvironment.id,
      createdTs: ts,
      lastUpdatedTs: ts,
      type: "bb.msg.environment.create",
      status: "DELIVERED",
      creatorId: FAKE_API_CALLER_ID,
      workspaceId: WORKSPACE_ID,
      payload: {
        environmentName: createdEnvironment.name,
      },
    };
    postMessageToOwnerAndDBA(schema, FAKE_API_CALLER_ID, messageTemplate);

    return createdEnvironment;
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
    let type = "bb.msg.environment.update";

    if (attrs.rowStatus && attrs.rowStatus != environment.rowStatus) {
      if (attrs.rowStatus == "ARCHIVED") {
        type = "bb.msg.environment.archive";
      } else if (attrs.rowStatus == "NORMAL") {
        type = "bb.msg.environment.restore";
      } else if (attrs.rowStatus == "PENDING_DELETE") {
        type = "bb.msg.environment.delete";
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
      lastUpdatedTs: ts,
      type,
      status: "DELIVERED",
      creatorId: FAKE_API_CALLER_ID,
      workspaceId: WORKSPACE_ID,
      payload: {
        environmentName: updatedEnvironment.name,
        changeList,
      },
    };
    postMessageToOwnerAndDBA(schema, FAKE_API_CALLER_ID, messageTemplate);

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
