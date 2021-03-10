/*
 * Mirage JS guide on Routes: https://miragejs.com/docs/route-handlers/functions
 */
import { Response } from "miragejs";
import { TaskBuiltinFieldId } from "../../plugins";

const WORKSPACE_ID = 1;

export default function routes() {
  // Change this value to simulate response delay.
  // By default development environment has a 400ms delay.
  this.timing = 0;

  this.namespace = "api";

  // User
  this.get("/user/:id");

  this.get("/user");

  this.post("/user", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("user");
    const user = schema.users.findBy({ email: attrs.email });
    if (user) {
      return new Response(200, {}, user);
    }
    return schema.users.create(attrs);
  });

  this.patch("/user/:userId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("user");
    return schema.users.find(request.params.userId).update(attrs);
  });

  // Auth
  this.post("/auth/login", function (schema, request) {
    const loginInfo = this.normalizedRequestAttrs("login-info");
    const user = schema.users.findBy({
      email: loginInfo.email,
      passwordHash: loginInfo.password,
    });
    if (user) {
      return user;
    }
    return new Response(
      401,
      {},
      { errors: loginInfo.email + " not found or incorrect password" }
    );
  });

  this.post("/auth/signup", function (schema, request) {
    const signupInfo = this.normalizedRequestAttrs("signup-info");
    const user = schema.users.findBy({ email: signupInfo.email });
    if (user) {
      return new Response(
        409,
        {},
        { errors: signupInfo.email + " already exists" }
      );
    }
    const ts = Date.now();
    const createdUser = schema.users.create({
      createdTs: ts,
      lastUpdatedTs: ts,
      status: "ACTIVE",
      ...signupInfo,
    });

    const newRoleMapping = {
      principalId: createdUser.id,
      email: createdUser.email,
      createdTs: ts,
      lastUpdatedTs: ts,
      role: "DEVELOPER",
      updaterId: createdUser.id,
      workspaceId: WORKSPACE_ID,
    };
    schema.roleMappings.create(newRoleMapping);

    return createdUser;
  });

  this.post("/auth/activate", function (schema, request) {
    const activateInfo = this.normalizedRequestAttrs("activate-info");
    if (!activateInfo.token) {
      return new Response(400, {}, { errors: "Missing activation token" });
    }

    const user = schema.users.findBy({ email: activateInfo.email });
    if (user) {
      const ts = Date.now();
      user.update({
        name: activateInfo.name,
        status: "ACTIVE",
        lastUpdatedTs: ts,
        passwordHash: activateInfo.password,
      });
      return user;
    }

    return new Response(
      400,
      {},
      { errors: activateInfo.email + " is not invited" }
    );
  });

  // RoleMapping
  this.get("/rolemapping", function (schema, request) {
    return schema.roleMappings.where((roleMapping) => {
      return roleMapping.workspaceId == WORKSPACE_ID;
    });
  });

  this.post("/rolemapping", function (schema, request) {
    const ts = Date.now();
    const attrs = {
      ...this.normalizedRequestAttrs("role-mapping"),
      workspaceId: WORKSPACE_ID,
    };
    const newRoleMapping = {
      ...attrs,
      createdTs: ts,
      lastUpdatedTs: ts,
      role: attrs.role,
      updaterId: attrs.updaterId,
      workspaceId: WORKSPACE_ID,
    };
    return schema.roleMappings.create(newRoleMapping);
  });

  this.patch("/rolemapping/:roleMappingId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("role-mapping");
    return schema.roleMappings.find(request.params.roleMappingId).update(attrs);
  });

  this.delete("/rolemapping/:roleMappingId", function (schema, request) {
    return schema.roleMappings.find(request.params.roleMappingId).destroy();
  });

  // Task
  this.get("/task", function (schema, request) {
    const {
      queryParams: { userid: userId },
    } = request;

    if (userId) {
      return schema.tasks.where((task) => {
        return (
          task.workspaceId == WORKSPACE_ID &&
          (task.creatorId == userId ||
            task.assigneeId == userId ||
            task.subscriberIdList.includes(userId))
        );
      });
    }
    return schema.tasks.none();
  });

  this.get("/task/:id", function (schema, request) {
    const task = schema.tasks.find(request.params.id);
    if (task) {
      return task;
    }
    return new Response(
      404,
      {},
      { errors: "Task " + request.params.id + " not found" }
    );
  });

  this.post("/task", function (schema, request) {
    const ts = Date.now();
    const attrs = this.normalizedRequestAttrs("task");
    const newTask = {
      ...attrs,
      createdTs: ts,
      lastUpdatedTs: ts,
      status: "OPEN",
      workspaceId: WORKSPACE_ID,
    };
    const createdTask = schema.tasks.create(newTask);

    schema.activities.create({
      createdTs: ts,
      lastUpdatedTs: ts,
      actionType: "bytebase.task.create",
      containerId: createdTask.id,
      creatorId: attrs.creatorId,
      workspaceId: WORKSPACE_ID,
    });

    return createdTask;
  });

  this.patch("/task/:taskId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("task-patch");
    const task = schema.tasks.find(request.params.taskId);
    if (task) {
      if (attrs.stageProgressList) {
        attrs.stageProgressList = task.stageProgressList.map((item) => {
          for (const stage of attrs.stageProgressList) {
            if (item.id === stage.id) {
              item.status = stage.status;
              break;
            }
          }
          return item;
        });
      }

      const changeList = [];

      if (attrs.status) {
        if (task.status != attrs.status) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.STATUS,
            oldValue: task.status,
            newValue: attrs.status,
          });
        }
      }

      if (attrs.assigneeId) {
        if (task.assigneeId != attrs.assigneeId) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.ASSIGNEE,
            oldValue: task.assigneeId,
            newValue: attrs.assigneeId,
          });
        }
      }

      // Empty string is valid
      if (attrs.description !== undefined) {
        if (task.description != attrs.description) {
          changeList.push({
            fieldId: TaskBuiltinFieldId.DESCRIPTION,
            oldValue: task.description,
            newValue: attrs.description,
          });
        }
      }

      for (const fieldId in attrs.payload) {
        const oldValue = task.payload[fieldId];
        const newValue = attrs.payload[fieldId];
        if (oldValue != newValue) {
          changeList.push({
            fieldId: fieldId,
            oldValue: task.payload[fieldId],
            newValue: attrs.payload[fieldId],
          });
        }
      }

      const ts = Date.now();
      const updatedTask = task.update({ ...attrs, lastUpdatedTs: ts });

      schema.activities.create({
        createdTs: ts,
        lastUpdatedTs: ts,
        actionType: "bytebase.task.field.update",
        containerId: updatedTask.id,
        creatorId: attrs.updaterId,
        payload: changeList.length > 0 ? { changeList } : undefined,
        workspaceId: WORKSPACE_ID,
      });

      return updatedTask;
    }
    return new Response(
      404,
      {},
      { errors: "Task " + request.params.id + " not found" }
    );
  });

  // environment
  this.get("/environment", function (schema, request) {
    return schema.environments
      .where((environment) => {
        return environment.workspaceId == WORKSPACE_ID;
      })
      .sort((a, b) => a.order - b.order);
  });

  this.post("/environment", function (schema, request) {
    let order = 0;
    const list = schema.environments.where((environment) => {
      return environment.workspaceId == WORKSPACE_ID;
    });
    if (list.length > 0) {
      order = list.sort((a, b) => b.order - a.order).models[0].order + 1;
    }
    const newEnvironment = {
      ...this.normalizedRequestAttrs("environment"),
      workspaceId: WORKSPACE_ID,
      order,
    };
    return schema.environments.create(newEnvironment);
  });

  this.patch("/environment/batch", function (schema, request) {
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
    return schema.environments
      .where((environment) => {
        return environment.workspaceId == WORKSPACE_ID;
      })
      .sort((a, b) => a.order - b.order);
  });

  this.patch("/environment/:environmentId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("environment");
    return schema.environments.find(request.params.environmentId).update(attrs);
  });

  this.delete("/environment/:environmentId", function (schema, request) {
    return schema.environments.find(request.params.environmentId).destroy();
  });

  // Instance
  this.get("/instance", function (schema, request) {
    return schema.instances.where((instance) => {
      return instance.workspaceId == WORKSPACE_ID;
    });
  });

  this.get("/instance/:id", function (schema, request) {
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

  this.post("/instance", function (schema, request) {
    const newInstance = {
      ...this.normalizedRequestAttrs("instance"),
      workspaceId: WORKSPACE_ID,
    };
    return schema.instances.create(newInstance);
  });

  this.patch("/instance/:instanceId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("instance");
    return schema.instances.find(request.params.instanceId).update(attrs);
  });

  this.delete("/instance/:instanceId", function (schema, request) {
    return schema.instances.find(request.params.instanceId).destroy();
  });

  // Data Source
  this.get("/instance/:instanceId/datasource", function (schema, request) {
    const instance = schema.instances.find(request.params.instanceId);
    if (instance) {
      return schema.dataSources.where((dataSource) => {
        return dataSource.instanceId == instance.id;
      });
    }
    return new Response(
      404,
      {},
      { errors: "Instance " + request.params.instanceId + " not found" }
    );
  });

  this.get("/instance/:instanceId/datasource/:id", function (schema, request) {
    const instance = schema.instances.find(request.params.instanceId);
    if (instance) {
      const dataSource = schema.dataSources.find(request.params.id);
      if (dataSource) {
        return dataSource;
      }
      return new Response(
        404,
        {},
        { errors: "Data Source " + request.params.id + " not found" }
      );
    }
    return new Response(
      404,
      {},
      { errors: "Instance " + request.params.instanceId + " not found" }
    );
  });

  this.post("/instance/:instanceId/datasource", function (schema, request) {
    const instance = schema.instances.find(request.params.instanceId);
    if (instance) {
      const newDataSource = {
        ...this.normalizedRequestAttrs("data-source"),
        instanceId: request.params.instanceId,
      };
      return schema.dataSources.create(newDataSource);
    }
    return new Response(
      404,
      {},
      { errors: "Instance " + request.params.instanceId + " not found" }
    );
  });

  this.patch(
    "/instance/:instanceId/datasource/:id",
    function (schema, request) {
      const instance = schema.instances.find(request.params.instanceId);
      if (instance) {
        const attrs = this.normalizedRequestAttrs("data-source");
        return schema.dataSources.find(request.params.id).update(attrs);
      }
      return new Response(
        404,
        {},
        { errors: "Instance " + request.params.instanceId + " not found" }
      );
    }
  );

  this.delete(
    "/instance/:instanceId/datasource/:id",
    function (schema, request) {
      const instance = schema.instances.find(request.params.instanceId);
      if (instance) {
        return schema.dataSources.find(request.params.id).destroy();
      }
      return new Response(
        404,
        {},
        { errors: "Instance " + request.params.instanceId + " not found" }
      );
    }
  );

  // Bookmark
  this.get("/bookmark", function (schema, request) {
    const {
      queryParams: { userid: userId },
    } = request;

    return schema.bookmarks.where({
      workspaceId: WORKSPACE_ID,
      creatorId: userId,
    });
  });

  this.post("/bookmark", function (schema, request) {
    const newBookmark = {
      ...this.normalizedRequestAttrs("bookmark"),
      workspaceId: WORKSPACE_ID,
    };
    return schema.bookmarks.create(newBookmark);
  });

  this.patch("/bookmark/:bookmarkId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("bookmark");
    return schema.bookmarks.find(request.params.bookmarkId).update(attrs);
  });

  this.delete("/bookmark/:bookmarkId", function (schema, request) {
    return schema.bookmarks.find(request.params.bookmarkId).destroy();
  });

  // Activity
  this.get("/activity", function (schema, request) {
    const {
      queryParams: { containerid: containerId, type },
    } = request;
    return schema.activities.where((activity) => {
      if (activity.workspaceId != WORKSPACE_ID) {
        return false;
      }

      if (containerId && containerId != activity.containerId) {
        return false;
      }

      if (type && !activity.actionType.startsWith(type)) {
        return false;
      }

      return true;
    });
  });

  this.post("/activity", function (schema, request) {
    const ts = Date.now();
    const attrs = this.normalizedRequestAttrs("activity");
    const newActivity = {
      ...attrs,
      createdTs: ts,
      lastUpdatedTs: ts,
      workspaceId: WORKSPACE_ID,
    };
    const createdActivity = schema.activities.create(newActivity);
    return createdActivity;
  });

  this.patch("/activity/:activityId", function (schema, request) {
    const attrs = this.normalizedRequestAttrs("activity-patch");
    const activity = schema.activities.find(request.params.activityId);
    if (activity) {
      return activity.update({ ...attrs, lastUpdatedTs: Date.now() });
    }
    return new Response(
      404,
      {},
      { errors: "Activity " + request.params.activityId + " not found" }
    );
  });

  this.delete("/activity/:activityId", function (schema, request) {
    return schema.activities.find(request.params.activityId).destroy();
  });
}
