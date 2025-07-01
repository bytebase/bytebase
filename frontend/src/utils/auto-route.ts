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
import { useAppFeature } from "@/store";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { formatEnvironmentName, type Environment } from "@/types/v1/environment";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto/v1/project_service";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "./v1";

export const isSQLEditorRoute = (router: Router) => {
  return router.currentRoute.value.name?.toString().startsWith("sql-editor");
};

// should go to SQL Editor route if
// - isSQLEditorRoute
// - && databaseChangeMode === EDITOR
export const shouldGoToSQLEditorRoute = (router: Router) => {
  const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");
  return (
    isSQLEditorRoute(router) &&
    databaseChangeMode.value === DatabaseChangeMode.EDITOR
  );
};

export const autoDatabaseRoute = (router: Router, database: Database) => {
  const name = shouldGoToSQLEditorRoute(router)
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
  if (shouldGoToSQLEditorRoute(router)) {
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
  if (shouldGoToSQLEditorRoute(router)) {
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
  if (shouldGoToSQLEditorRoute(router)) {
    return {
      name: SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
      hash: `#${environment.id}`,
    };
  }
  return {
    path: `/${formatEnvironmentName(environment.id)}`,
  };
};

export const autoSubscriptionRoute = (router: Router) => {
  if (shouldGoToSQLEditorRoute(router)) {
    return { name: SQL_EDITOR_SETTING_SUBSCRIPTION_MODULE };
  }
  return { name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION };
};

export const autoProfileLink = (router: Router, user?: User) => {
  if (shouldGoToSQLEditorRoute(router)) {
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
