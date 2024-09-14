import { type Router } from "vue-router";
import { INSTANCE_ROUTE_DETAIL } from "@/router/dashboard/instance";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
} from "@/router/dashboard/projectV1";
import {
  SETTING_ROUTE_PROFILE,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/router/dashboard/workspaceSetting";
import {
  SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE,
  SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
  SQL_EDITOR_SETTING_INSTANCE_MODULE,
  SQL_EDITOR_SETTING_PROFILE_MODULE,
  SQL_EDITOR_SETTING_PROJECT_MODULE,
  SQL_EDITOR_SETTING_SUBSCRIPTION_MODULE,
  SQL_EDITOR_SETTING_USERS_MODULE,
} from "@/router/sqlEditor";
import type { User } from "@/types/proto/v1/auth_service";
import type { Database } from "@/types/proto/v1/database_service";
import type { Environment } from "@/types/proto/v1/environment_service";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto/v1/instance_service";
import type { Project } from "@/types/proto/v1/project_service";
import {
  extractDatabaseResourceName,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "./v1";

export const isSQLEditorRoute = (router: Router) => {
  return router.currentRoute.value.name?.toString().startsWith("sql-editor");
};

export const autoDatabaseRoute = (router: Router, database: Database) => {
  const name = isSQLEditorRoute(router)
    ? SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE
    : PROJECT_V1_ROUTE_DATABASE_DETAIL;

  const projectId = extractProjectResourceName(database.project);
  const { instanceName: instanceId, databaseName } =
    extractDatabaseResourceName(database.name);
  return {
    name,
    params: {
      projectId,
      instanceId,
      databaseName,
    },
  };
};

export const autoInstanceRoute = (
  router: Router,
  instance: Instance | InstanceResource
) => {
  if (isSQLEditorRoute(router)) {
    return {
      name: SQL_EDITOR_SETTING_INSTANCE_MODULE,
      hash: `#${instance.name}`,
    };
  }
  return {
    name: INSTANCE_ROUTE_DETAIL,
    params: {
      instanceId: extractInstanceResourceName(instance.name),
    },
  };
};

export const autoProjectRoute = (router: Router, project: Project) => {
  if (isSQLEditorRoute(router)) {
    return {
      name: SQL_EDITOR_SETTING_PROJECT_MODULE,
      hash: `#${project.name}`,
    };
  }
  return {
    name: PROJECT_V1_ROUTE_DATABASES,
    params: {
      projectId: extractProjectResourceName(project.name),
    },
  };
};

export const autoEnvironmentRoute = (
  router: Router,
  environment: Environment
) => {
  if (isSQLEditorRoute(router)) {
    return {
      name: SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
      hash: `#${extractEnvironmentResourceName(environment.name)}`,
    };
  }
  return {
    path: `/${environment.name}`,
  };
};

export const autoSubscriptionRoute = (router: Router) => {
  if (isSQLEditorRoute(router)) {
    return { name: SQL_EDITOR_SETTING_SUBSCRIPTION_MODULE };
  }
  return { name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION };
};

export const autoProfileLink = (router: Router, user?: User) => {
  if (isSQLEditorRoute(router)) {
    if (user) {
      return {
        name: SQL_EDITOR_SETTING_USERS_MODULE,
        hash: `#${user.email}`,
      };
    }
    return {
      name: SQL_EDITOR_SETTING_PROFILE_MODULE,
    };
  }
  if (user) {
    return {
      path: `/users/${user.email}`,
    };
  }
  return {
    name: SETTING_ROUTE_PROFILE,
  };
};
