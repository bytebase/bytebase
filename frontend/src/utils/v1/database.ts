import { orderBy } from "lodash-es";
import { SimpleExpr } from "@/plugins/cel";
import {
  hasFeature,
  useCurrentUserIamPolicy,
  useSubscriptionV1Store,
} from "@/store";
import { ComposedDatabase, UNKNOWN_ID } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { Engine, State } from "@/types/proto/v1/common";
import { DataSourceType } from "@/types/proto/v1/instance_service";
import {
  hasPermissionToCreateChangeDatabaseIssue,
  hasProjectPermissionV2,
  hasWorkspaceLevelProjectPermission,
} from "../iam";
import { isDev, semverCompare } from "../util";
import {
  extractProjectResourceName,
  isDeveloperOfProjectV1,
  isOwnerOfProjectV1,
} from "./project";

export const databaseV1Url = (db: ComposedDatabase) => {
  const project = extractProjectResourceName(db.project);
  const { instance, database } = extractDatabaseResourceName(db.name);
  return `/projects/${encodeURIComponent(
    project
  )}/instances/${encodeURIComponent(instance)}/databases/${encodeURIComponent(
    database
  )}`;
};

export const extractDatabaseResourceName = (
  resource: string
): {
  instance: string;
  database: string;
  full: string;
} => {
  const pattern =
    /(?:^|\/)instances\/(?<instance>[^/]+)\/databases\/(?<database>[^/]+)(?:$|\/)/;
  const matches = resource.match(pattern);
  if (matches) {
    const { instance = String(UNKNOWN_ID), database = "" } =
      matches.groups ?? {};
    return {
      instance,
      database,
      full: `instances/${instance}/databases/${database}`,
    };
  }
  return {
    instance: String(UNKNOWN_ID),
    database: "",
    full: `instances/${UNKNOWN_ID}/databases/`,
  };
};

export const sortDatabaseV1List = (databaseList: ComposedDatabase[]) => {
  return orderBy(
    databaseList,
    [
      (db) => db.effectiveEnvironmentEntity.order,
      (db) => Number(db.instanceEntity.uid),
      (db) => Number(db.projectEntity.uid),
      (db) => db.databaseName,
    ],
    ["desc", "asc", "asc", "asc"]
  );
};

export const isPITRDatabaseV1 = (db: ComposedDatabase): boolean => {
  // A pitr database's name is xxx_pitr_1234567890 or xxx_pitr_1234567890_del
  return !!db.databaseName.match(/^(.+?)_pitr_(\d+)(_del)?$/);
};

export const isArchivedDatabaseV1 = (db: ComposedDatabase): boolean => {
  if (db.instanceEntity.state === State.DELETED) {
    return true;
  }
  if (db.effectiveEnvironmentEntity.state === State.DELETED) {
    return true;
  }
  return false;
};

// isDatabaseV1Alterable checks if database alterable for user.
export const isDatabaseV1Alterable = (
  database: ComposedDatabase,
  user: User
): boolean => {
  if (!hasFeature("bb.feature.access-control")) {
    // The current plan doesn't have access control feature.
    // Fallback to true.
    return true;
  }

  if (hasPermissionToCreateChangeDatabaseIssue(database, user)) {
    return true;
  }

  // If user is owner or developer of its projects, we will show the database in the UI.
  if (
    isOwnerOfProjectV1(database.projectEntity.iamPolicy, user) ||
    isDeveloperOfProjectV1(database.projectEntity.iamPolicy, user)
  ) {
    return true;
  }

  return false;
};

// isDatabaseV1Queryable checks if database allowed to query in SQL Editor.
export const isDatabaseV1Queryable = (
  database: ComposedDatabase,
  user: User
): boolean => {
  if (!hasFeature("bb.feature.access-control")) {
    // The current plan doesn't have access control feature.
    // Fallback to true.
    return true;
  }

  if (hasWorkspaceLevelProjectPermission(user, "bb.databases.query")) {
    return true;
  }

  const currentUserIamPolicy = useCurrentUserIamPolicy();
  if (currentUserIamPolicy.allowToQueryDatabaseV1(database)) {
    return true;
  }

  // denied otherwise
  return false;
};

// isTableQueryable checks if table allowed to query in SQL Editor.
export const isTableQueryable = (
  database: ComposedDatabase,
  schema: string,
  table: string,
  user: User
): boolean => {
  if (hasWorkspaceLevelProjectPermission(user, "bb.databases.query")) {
    // The current user has the super privilege to access all databases.
    // AKA. Owners and DBAs
    return true;
  }

  const currentUserIamPolicy = useCurrentUserIamPolicy();
  if (currentUserIamPolicy.allowToQueryDatabaseV1(database, schema, table)) {
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
    db.effectiveEnvironmentEntity.title.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (columns.includes("tenant")) {
    const tenantValue = db.labels["tenant"] ?? "";
    if (tenantValue.toLowerCase().includes(keyword)) {
      return true;
    }
  }

  return false;
}

export const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.6.0";

export function allowGhostMigrationV1(
  databaseList: ComposedDatabase[]
): boolean {
  const subscriptionV1Store = useSubscriptionV1Store();
  return databaseList.every((db) => {
    return (
      db.instanceEntity.engine === Engine.MYSQL &&
      subscriptionV1Store.hasInstanceFeature(
        "bb.feature.online-migration",
        db.instanceEntity
      ) &&
      semverCompare(
        db.instanceEntity.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION,
        "gte"
      )
    );
  });
}

export function allowDatabaseV1Access(
  database: ComposedDatabase,
  user: User,
  type: DataSourceType
): boolean {
  // "ADMIN" data source should only be used by the system, thus it shouldn't
  // touch this method at all. If it touches somehow, we will reject it and
  // log a warning
  if (type === DataSourceType.ADMIN) {
    if (isDev()) {
      console.trace(
        "Should not check database access against ADMIN connection"
      );
    } else {
      console.warn("Should not check database access against ADMIN connection");
    }
    return false;
  }

  if (
    hasProjectPermissionV2(database.projectEntity, user, "bb.databases.get")
  ) {
    return true;
  }

  return false;
}

export const extractEnvironmentNameListFromExpr = (
  expr: SimpleExpr
): string[] => {
  const [left, right] = expr.args;
  if (expr.operator === "@in" && left === "resource.environment_name") {
    return right as any as string[];
  }
  return [];
};
