import { keyBy } from "lodash-es";
import slug from "slug";

import {
  ComposedDatabase,
  unknownEnvironment,
  UNKNOWN_ID,
  UNKNOWN_INSTANCE_NAME,
} from "@/types";
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import { instanceV1Slug } from "./instance";
import { Policy, PolicyType } from "@/types/proto/v1/org_policy_service";
import { User } from "@/types/proto/v1/auth_service";
import {
  hasFeature,
  useCurrentUserIamPolicy,
  useEnvironmentV1Store,
} from "@/store";
import { hasWorkspacePermissionV1 } from "../role";
import { Instance } from "@/types/proto/v1/instance_service";
import { Engine, State } from "@/types/proto/v1/common";
import { semverCompare } from "../util";

export const databaseV1Slug = (db: ComposedDatabase) => {
  return [slug(db.databaseName), db.uid].join("-");
};

export function connectionV1Slug(
  instance: Instance,
  database?: ComposedDatabase
): string {
  const parts = [instanceV1Slug(instance)];
  if (database && database.uid !== String(UNKNOWN_ID)) {
    parts.push(databaseV1Slug(database));
  }
  return parts.join("_");
}

export const extractDatabaseResourceName = (
  resource: string
): {
  instance: string /** Format: instances/{instance} */;
  database: string;
} => {
  const pattern =
    /^(?<instance>instances\/[^/]+)\/databases\/(?<database>[^/]+)$/;
  const matches = resource.match(pattern);
  if (matches) {
    const { instance = UNKNOWN_INSTANCE_NAME, database = "" } =
      matches.groups ?? {};
    return {
      instance,
      database,
    };
  }
  return {
    instance: UNKNOWN_INSTANCE_NAME,
    database: "",
  };
};

export const sortDatabaseV1ListByEnvironmentV1 = (
  databaseList: ComposedDatabase[],
  environmentList: Environment[]
) => {
  const environmentMap = keyBy(environmentList, (env) => env.name);
  return databaseList.sort((a, b) => {
    const aEnv = environmentMap[a.instanceEntity.environment];
    const bEnv = environmentMap[b.instanceEntity.environment];
    const aEnvOrder = aEnv?.order ?? -1;
    const bEnvOrder = bEnv?.order ?? -1;

    return bEnvOrder - aEnvOrder;
  });
};

export const isPITRDatabaseV1 = (db: ComposedDatabase): boolean => {
  // A pitr database's name is xxx_pitr_1234567890 or xxx_pitr_1234567890_del
  return !!db.databaseName.match(/^(.+?)_pitr_(\d+)(_del)?$/);
};

export const isArchivedDatabaseV1 = (db: ComposedDatabase): boolean => {
  if (db.instanceEntity.state === State.DELETED) {
    return true;
  }
  if (db.instanceEntity.environmentEntity.state === State.DELETED) {
    return true;
  }
  return false;
};

export const isDatabaseV1Accessible = (
  database: ComposedDatabase,
  policyList: Policy[],
  user: User
): boolean => {
  if (!hasFeature("bb.feature.access-control")) {
    // The current plan doesn't have access control feature.
    // Fallback to true.
    return true;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-access-control",
      user.userRole
    )
  ) {
    // The current user has the super privilege to access all databases.
    // AKA. Owners and DBAs
    return true;
  }

  const environment =
    useEnvironmentV1Store().getEnvironmentByName(
      database.instanceEntity.environment
    ) ?? unknownEnvironment();
  if (environment.tier === EnvironmentTier.UNPROTECTED) {
    return true;
  }

  const policy = policyList.find((policy) => {
    const { type, resourceUid, enforce } = policy;
    return (
      type === PolicyType.ACCESS_CONTROL &&
      resourceUid === `${database.uid}` &&
      enforce
    );
  });
  if (policy) {
    // The database is in the allowed list
    return true;
  }
  const currentUserIamPolicy = useCurrentUserIamPolicy();
  if (currentUserIamPolicy.allowToQueryDatabaseV1(database)) {
    return true;
  }

  // denied otherwise
  return false;
};

type DatabaseV1FilterFields =
  | "name"
  | "project"
  | "instance"
  | "environment"
  | "tenant";
export function filterDatabaseV1ByKeyword(
  db: ComposedDatabase,
  keyword: string,
  columns: DatabaseV1FilterFields[] = ["name"]
): boolean {
  keyword = keyword.trim().toLowerCase();
  if (!keyword) {
    // Skip the filter
    return true;
  }

  if (
    columns.includes("name") &&
    db.databaseName.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("project") &&
    db.projectEntity.title.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("instance") &&
    db.instanceEntity.title.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("environment") &&
    db.instanceEntity.environmentEntity.title.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (columns.includes("tenant")) {
    const tenantValue = db.labels["bb.tenant"] ?? "";
    if (tenantValue.toLowerCase().includes(keyword)) {
      return true;
    }
  }

  return false;
}

const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.7.0";

export function allowGhostMigrationV1(
  databaseList: ComposedDatabase[]
): boolean {
  return databaseList.every((db) => {
    return (
      db.instanceEntity.engine === Engine.MYSQL &&
      semverCompare(
        db.instanceEntity.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION,
        "gte"
      )
    );
  });
}
