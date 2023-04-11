import type {
  Database,
  DataSourceType,
  Environment,
  Principal,
} from "../types";
import { hasWorkspacePermission } from "./role";
import { isDev, semverCompare } from "./util";

export function allowDatabaseAccess(
  database: Database,
  principal: Principal,
  type: DataSourceType
): boolean {
  // "ADMIN" data source should only be used by the system, thus it shouldn't
  // touch this method at all. If it touches somehow, we will reject it and
  // log a warning
  if (type == "ADMIN") {
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
    hasWorkspacePermission(
      "bb.permission.workspace.manage-instance",
      principal.role
    )
  ) {
    return true;
  }

  return false;
}

// Sort the list to put prod items first.
export function sortDatabaseList(
  list: Database[],
  environmentList: Environment[]
): Database[] {
  return list.sort((a: Database, b: Database) => {
    let aEnvIndex = -1;
    let bEnvIndex = -1;

    for (let i = 0; i < environmentList.length; i++) {
      if (environmentList[i].id == a.instance.environment.id) {
        aEnvIndex = i;
      }
      if (environmentList[i].id == b.instance.environment.id) {
        bEnvIndex = i;
      }

      if (aEnvIndex != -1 && bEnvIndex != -1) {
        break;
      }
    }
    return bEnvIndex - aEnvIndex;
  });
}

const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.7.0";

export function allowGhostMigration(databaseList: Database[]): boolean {
  return databaseList.every((db) => {
    return (
      db.instance.engine === "MYSQL" &&
      semverCompare(db.instance.engineVersion, MIN_GHOST_SUPPORT_MYSQL_VERSION)
    );
  });
}

type DatabaseFilterFields =
  | "name"
  | "project"
  | "instance"
  | "environment"
  | "tenant";
export function filterDatabaseByKeyword(
  db: Database,
  keyword: string,
  columns: DatabaseFilterFields[] = ["name"]
): boolean {
  keyword = keyword.trim().toLowerCase();
  if (!keyword) {
    // Skip the filter
    return true;
  }

  if (columns.includes("name") && db.name.toLowerCase().includes(keyword)) {
    return true;
  }

  if (
    columns.includes("project") &&
    db.project.name.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("instance") &&
    db.instance.name.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("environment") &&
    db.instance.environment.name.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("tenant") &&
    db.labels
      .find((label) => label.key === "bb.tenant")
      ?.value.toLowerCase()
      .includes(keyword)
  ) {
    return true;
  }

  return false;
}

export function isPITRDatabase(db: Database): boolean {
  const { name } = db;
  // A pitr database's name is xxx_pitr_1234567890 or xxx_pitr_1234567890_del
  return !!name.match(/^(.+?)_pitr_(\d+)(_del)?$/);
}

export function isArchivedDatabase(db: Database): boolean {
  if (db.instance.rowStatus === "ARCHIVED") {
    return true;
  }
  if (db.instance.environment.rowStatus === "ARCHIVED") {
    return true;
  }
  return false;
}

export const extractDatabaseResourceName = (
  resource: string
): {
  environment?: string;
  instance?: string;
  database?: string;
} => {
  const pattern =
    /^environments\/([^/]+)\/instances\/([^/]+)\/databases\/([^/]+)$/;
  const matches = resource.match(pattern);
  if (matches) {
    const [_, environment, instance, database] = matches;
    return {
      environment,
      instance,
      database,
    };
  }
  return {
    environment: undefined,
    instance: undefined,
    database: undefined,
  };
};
