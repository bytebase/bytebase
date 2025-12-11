import { head } from "lodash-es";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import { extractUserId } from "@/store/modules/v1/common";
import {
  type ComposedDatabase,
  type SQLEditorConnection,
  UNKNOWN_PROJECT_NAME,
} from "@/types";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import { Worksheet_Visibility } from "@/types/proto-es/v1/worksheet_service_pb";
import {
  emptySQLEditorConnection,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
  isDatabaseV1Queryable,
} from "@/utils";

export const extractWorksheetUID = (name: string) => {
  const pattern = /(?:^|\/)worksheets\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "-1";
};

// readable to
// PRIVATE: workspace Owner/DBA and the creator only.
// PROJECT_WRITE: workspace Owner/DBA and all members in the project.
// PROJECT_READ: workspace Owner/DBA and all members in the project.
export const isWorksheetReadableV1 = (sheet: Worksheet) => {
  const currentUser = useCurrentUserV1();

  if (extractUserId(sheet.creator) === currentUser.value.email) {
    // Always readable to the creator
    return true;
  }

  if (hasWorkspacePermissionV2("bb.worksheets.manage")) {
    return true;
  }

  switch (sheet.visibility) {
    case Worksheet_Visibility.PRIVATE:
      return false;
    case Worksheet_Visibility.PROJECT_READ:
    case Worksheet_Visibility.PROJECT_WRITE: {
      const projectV1 = useProjectV1Store().getProjectByName(sheet.project);
      if (projectV1.name === UNKNOWN_PROJECT_NAME) {
        return false;
      }
      return hasProjectPermissionV2(projectV1, "bb.worksheets.get");
    }
  }
  return false;
};

// writable to
// PRIVATE: workspace Owner/DBA and the creator only.
// PROJECT_WRITE: workspace Owner/DBA and all members in the project.
// PROJECT_READ: workspace Owner/DBA and project owner.
export const isWorksheetWritableV1 = (sheet: Worksheet) => {
  const currentUser = useCurrentUserV1();

  if (extractUserId(sheet.creator) === currentUser.value.email) {
    // Always writable to the creator
    return true;
  }

  if (hasWorkspacePermissionV2("bb.worksheets.manage")) {
    return true;
  }

  const projectV1 = useProjectV1Store().getProjectByName(sheet.project);
  if (projectV1.name === UNKNOWN_PROJECT_NAME) {
    return false;
  }
  switch (sheet.visibility) {
    case Worksheet_Visibility.PRIVATE:
      return false;
    case Worksheet_Visibility.PROJECT_WRITE:
      return hasProjectPermissionV2(projectV1, "bb.projects.get");
    case Worksheet_Visibility.PROJECT_READ:
      return hasProjectPermissionV2(projectV1, "bb.worksheets.manage");
  }

  return false;
};

export const setDefaultDataSourceForConn = (
  conn: SQLEditorConnection,
  database: ComposedDatabase
) => {
  if (conn.dataSourceId) {
    return;
  }

  // Default connect to the first read-only data source if available.
  // Skip checking env/project policy for now.
  const defaultDataSource =
    head(
      database.instanceResource.dataSources.filter(
        (d) => d.type === DataSourceType.READ_ONLY
      )
    ) || head(database.instanceResource.dataSources);
  if (defaultDataSource) {
    conn.dataSourceId = defaultDataSource.id;
  }
};

export const extractWorksheetConnection = async (worksheet: {
  database: string;
}) => {
  const connection = emptySQLEditorConnection();
  if (worksheet.database) {
    try {
      const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
        worksheet.database
      );
      if (!isDatabaseV1Queryable(database)) {
        return connection;
      }
      connection.instance = database.instance;
      connection.database = database.name;
      setDefaultDataSourceForConn(connection, database);
    } catch {
      // Skip.
    }
  }
  return connection;
};
