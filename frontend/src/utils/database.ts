import { groupBy } from "lodash-es";
import semverCompare from "semver/functions/compare";
import { Database, DataSourceType, Environment, Principal } from "../types";
import { isDBAOrOwner } from "./role";
import { isDev } from "./util";

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

  if (isDBAOrOwner(principal.role)) {
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

const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.6.0";

export function allowGhostMigration(databaseList: Database[]): boolean {
  const groupByEnvironment = groupBy(
    databaseList,
    (db) => db.instance.environment.id
  );
  // Multiple tasks in one stage is not supported by gh-ost now.
  for (const environmentId in groupByEnvironment) {
    const databaseListInStage = groupByEnvironment[environmentId];
    if (databaseListInStage.length > 1) {
      return false;
    }
  }

  return databaseList.every((db) => {
    return (
      db.instance.engine === "MYSQL" &&
      semverCompare(
        db.instance.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION
      ) >= 0
    );
  });
}

export function isPITRDatabase(db: Database): boolean {
  const { name } = db;
  // A pitr database's name is xxx_pitr_1234567890 or xxx_pitr_1234567890_del
  return !!name.match(/^(.+?)_pitr_(\d+)(_del)?$/);
}
