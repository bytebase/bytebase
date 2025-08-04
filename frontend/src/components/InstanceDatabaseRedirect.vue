<script setup lang="ts">
import { onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { INSTANCE_ROUTE_DETAIL } from "@/router/dashboard/instance";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "@/router/dashboard/projectV1";
import {
  databaseNamePrefix,
  instanceNamePrefix,
  pushNotification,
  useDatabaseV1Store,
} from "@/store";
import { isValidProjectName } from "@/types";
import { extractProjectResourceName } from "@/utils/v1";

const route = useRoute();
const router = useRouter();
const databaseStore = useDatabaseV1Store();

onMounted(async () => {
  const instanceId = route.params.instanceId as string;
  const databaseName = route.params.databaseName as string;

  try {
    // Fetch the database to get its project information
    const database = await databaseStore.getOrFetchDatabaseByName(
      `${instanceNamePrefix}${instanceId}/${databaseNamePrefix}${databaseName}`
    );

    if (database && isValidProjectName(database.project)) {
      // Extract project ID from the database's project field
      const projectId = extractProjectResourceName(database.project);

      // Redirect to the project database detail page
      router.replace({
        name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
        params: {
          projectId: projectId,
          instanceId: instanceId,
          databaseName: databaseName,
        },
      });
    } else {
      // If database not found, show error and redirect to instance detail
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: "Database not found",
        description: `Database: ${databaseName}`,
      });

      router.replace({
        name: INSTANCE_ROUTE_DETAIL,
        params: {
          instanceId: instanceId,
        },
      });
    }
  } catch (error) {
    console.error("Failed to fetch database:", error);

    // Show error notification
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Error",
      description: `Failed to load database: ${databaseName}`,
    });

    // On error, redirect to instance detail
    router.replace({
      name: INSTANCE_ROUTE_DETAIL,
      params: {
        instanceId: instanceId,
      },
    });
  }
});
</script>
