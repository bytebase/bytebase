import { Response } from "miragejs";
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
      { errors: "Database " + request.params.id + " not found." }
    );
  });

  route.get("/database", function (schema, request) {
    const {
      queryParams: {
        environment: environmentId,
        instance: instanceId,
        project: projectId,
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
        if (database.name == ALL_DATABASE_NAME) {
          return false;
        }

        if (instanceIdList && !instanceIdList.includes(database.instanceId)) {
          return false;
        }

        if (projectId && projectId != database.projectId) {
          return false;
        }

        if (userId) {
          const project = schema.projects.find(database.projectId);

          if (!project) {
            return false;
          }

          return (
            schema.projectMembers.findBy({
              principalId: userId,
              projectId: project.id,
              workspaceId: WORKSPACE_ID,
            }) != undefined
          );
        }

        return true;
      })
      .sort((a, b) =>
        a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
      );
  });

  route.post("/database", function (schema, request) {
    const ts = Date.now();
    const { issueId, creatorId, ...attrs } = this.normalizedRequestAttrs(
      "database-new"
    );

    if (
      schema.databases.findBy({
        name: attrs.name,
        instanceId: attrs.instanceId,
        workspaceId: WORKSPACE_ID,
      })
    ) {
      return new Response(
        409,
        {},
        {
          errors: `Database name ${attrs.name} already exists.`,
        }
      );
    }

    const newDatabase = {
      ...attrs,
      creatorId,
      createdTs: ts,
      updaterId: creatorId,
      updatedTs: ts,
      syncStatus: "OK",
      lastSuccessfulSyncTs: ts,
      workspaceId: WORKSPACE_ID,
    };

    if (issueId) {
      schema.activities.create({
        creatorId: creatorId,
        createdTs: ts,
        updaterId: creatorId,
        updatedTs: ts,
        actionType: "bb.issue",
        containerId: issueId,
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
    if (!database) {
      return new Response(
        404,
        {},
        { errors: "Database " + request.params.databaseId + " not found." }
      );
    }

    if (database.name == ALL_DATABASE_NAME) {
      return new Response(400, {}, { errors: "Can't update database *" });
    }

    return database.update({ ...attrs, updatedTs: Date.now() });
  });
}
