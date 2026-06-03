import { useEffect } from "react";
import {
  databaseNamePrefix,
  getProjectName,
  instanceNamePrefix,
  isValidProjectName,
} from "@/react/lib/resourceName";
import { router } from "@/react/router";
import {
  INSTANCE_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
} from "@/react/router/handles";
import { getOrFetchDatabaseByName } from "@/react/stores/app/databaseAccess";
import { pushNotification } from "@/store";

export function InstanceDatabaseRedirectPage({
  instanceId,
  databaseName,
}: {
  instanceId: string;
  databaseName: string;
}) {
  useEffect(() => {
    let active = true;

    const redirect = async () => {
      try {
        const database = await getOrFetchDatabaseByName(
          `${instanceNamePrefix}${instanceId}/${databaseNamePrefix}${databaseName}`
        );
        if (!active) {
          return;
        }
        if (isValidProjectName(database.project)) {
          void router.replace({
            name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
            params: {
              projectId: getProjectName(database.project),
              instanceId,
              databaseName,
            },
          });
          return;
        }
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: "Database not found",
          description: `Database: ${databaseName}`,
        });
      } catch (error) {
        if (!active) {
          return;
        }
        console.error("Failed to fetch database:", error);
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Error",
          description: `Failed to load database: ${databaseName}`,
        });
      }

      void router.replace({
        name: INSTANCE_ROUTE_DETAIL,
        params: {
          instanceId,
        },
      });
    };

    void redirect();

    return () => {
      active = false;
    };
  }, [databaseName, instanceId]);

  return null;
}
