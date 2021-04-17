import { Store } from "vuex";
import {
  Database,
  DataSourceMember,
  DataSourceType,
  Principal,
} from "../types";
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

  for (const dataSource of database.dataSourceList) {
    if (
      // Returns true if the current user has equal or higher access
      (dataSource.type == type || (dataSource.type == "RW" && type == "RO")) &&
      dataSource.memberList.find((item: DataSourceMember) => {
        return item.principal.id == principal.id;
      })
    ) {
      return true;
    }
  }
  return false;
}
