import { useEffect } from "react";
import { router } from "@/app/router";
import { INSTANCE_ROUTE_DETAIL } from "@/app/router/handles";
import {
  databaseNamePrefix,
  instanceNamePrefix,
  isValidProjectName,
} from "@/lib/resourceName";
import { pushNotification } from "@/stores";
import { getOrFetchDatabaseByName } from "@/stores/app/databaseAccess";
import { autoDatabaseRoute } from "@/utils";

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
          void router.replace(autoDatabaseRoute(database));
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
