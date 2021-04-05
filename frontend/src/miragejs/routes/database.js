import { WORKSPACE_ID } from "./index";
import { ALL_DATABASE_NAME } from "../../types";

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
      queryParams: { environment: environmentId },
    } = request;
    const instanceIdList = schema.instances
      .where({ workspaceId: WORKSPACE_ID, environmentId })
      .models.map((instance) => instance.id);
    if (instanceIdList.length == 0) {
      return [];
    }
    return schema.databases
      .where((database) => {
        // If environment is specified, then we don't include the database representing all databases,
        // since the all databases is per instance.
        if (environmentId && database.name == ALL_DATABASE_NAME) {
          return false;
        }
        return instanceIdList.includes(database.instanceId);
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

  route.get("/instance/:instanceId/database", function (schema, request) {
    const instance = schema.instances.find(request.params.instanceId);
    if (instance) {
      return schema.databases
        .where((database) => {
          return database.instanceId == instance.id;
        })
        .sort((a, b) =>
          a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
        );
    }
    return new Response(
      404,
      {},
      { errors: "Instance " + request.params.instanceId + " not found" }
    );
  });
}
