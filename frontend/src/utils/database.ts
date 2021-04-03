import { Store } from "vuex";
import {
  Database,
  DataSourceMember,
  DataSourceType,
  Principal,
} from "../types";

export function allowDatabaseAccess(
  database: Database,
  principal: Principal,
  type: DataSourceType
): boolean {
  if (principal.role == "DBA" || principal.role == "OWNER") {
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
