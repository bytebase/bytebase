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
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  type Environment,
  formatEnvironmentName,
} from "@/types/v1/environment";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
} from "./v1";

export const isSQLEditorRoute = (router: Router) => {
  return router.currentRoute.value.name?.toString().startsWith("sql-editor");
};

export const autoDatabaseRoute = (_: Router, database: Database) => {
  const name = PROJECT_V1_ROUTE_DATABASE_DETAIL;

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
  _: Router,
  instance: Instance | InstanceResource
) => {
  return {
    name: INSTANCE_ROUTE_DETAIL,
    params: {
      instanceId: extractInstanceResourceName(instance.name),
    },
  };
};

export const autoProjectRoute = (_: Router, project: Project) => {
  return {
    name: PROJECT_V1_ROUTE_DATABASES,
    params: {
      projectId: extractProjectResourceName(project.name),
    },
  };
};

export const autoEnvironmentRoute = (_: Router, environment: Environment) => {
  return {
    path: `/${formatEnvironmentName(environment.id)}`,
  };
};

export const autoSubscriptionRoute = (_: Router) => {
  return { name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION };
};

export const autoProfileLink = (_: Router, user?: User) => {
  if (user) {
    return {
      path: `/users/${user.email}`,
    };
  }
  return {
    name: SETTING_ROUTE_PROFILE,
  };
};
