import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";

export default function configureDatabase(route) {
  route.get("/database/:id", function (schema, request) {
    const database = schema.databases.find(request.params.id);
    if (database) {
      return database;
    }
    return new Response(
      404,
      {},
      { errors: "Database " + request.params.id + " not found" }
    );
  });

  route.get("/database", function (schema, request) {
    const {
      queryParams: {
        environment: environmentId,
        instance: instanceId,
        user: userId,
      },
    } = request;
    const instanceIdList = instanceId
      ? [instanceId]
      : environmentId
      ? schema.instances
          .where({ workspaceId: WORKSPACE_ID, environmentId })
          .models.map((instance) => instance.id)
      : schema.instances
          .where({ workspaceId: WORKSPACE_ID })
          .models.map((instance) => instance.id);
    if (instanceIdList && instanceIdList.length == 0) {
      return [];
    }

    return schema.databases
      .where((database) => {
        if (instanceIdList && !instanceIdList.includes(database.instanceId)) {
          return false;
        }

        if (userId) {
          const dataSourceList = schema.dataSources.where({
            workspaceId: WORKSPACE_ID,
            databaseId: database.id,
          });

          let matchFound = false;
          for (const dataSource of dataSourceList.models) {
            if (
              dataSource.memberList.find((item) => {
                return item.principalId == userId;
              })
            ) {
              matchFound = true;
              break;
            }
          }
          if (!matchFound) {
            return false;
          }
        }

        return true;
      })
      .sort((a, b) =>
        a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
      );
  });

  route.post("/database", function (schema, request) {
    const ts = Date.now();
    const { taskId, creatorId, ...attrs } = this.normalizedRequestAttrs(
      "database"
    );
    const newDatabase = {
      ...attrs,
      createdTs: ts,
      lastUpdatedTs: ts,
      syncStatus: "OK",
      lastSuccessfulSyncTs: ts,
      workspaceId: WORKSPACE_ID,
    };

    if (taskId) {
      schema.activities.create({
        createdTs: ts,
        lastUpdatedTs: ts,
        actionType: "bytebase.task",
        containerId: taskId,
        creatorId: creatorId,
        comment: `Created database ${newDatabase.name}`,
        workspaceId: WORKSPACE_ID,
      });
    }
    const createdDatabase = schema.databases.create(newDatabase);
    return createdDatabase;
  });

  route.patch("/database/:databaseId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("database-patch");
    const database = schema.databases.find(request.params.databaseId);
    if (database) {
      return database.update({ ...attrs, lastUpdatedTs: Date.now() });
    }
    return new Response(
      404,
      {},
      { errors: "Database " + request.params.databaseId + " not found" }
    );
  });
}
