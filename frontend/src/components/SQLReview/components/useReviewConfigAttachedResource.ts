import { type Ref, watchEffect, computed } from "vue";
import { t } from "@/plugins/i18n";
import {
  useEnvironmentV1Store,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import {
  environmentNamePrefix,
  projectNamePrefix,
  isDatabaseName,
} from "@/store/modules/v1/common";
import { hasWorkspacePermissionV2 } from "@/utils";

export type ResourceType = "environment" | "project" | "database";

export const useReviewConfigAttachedResource = (resource: Ref<string>) => {
  const environmentV1Store = useEnvironmentV1Store();
  const databaseStore = useDatabaseV1Store();
  const projectStore = useProjectV1Store();

  const resourceType = computed((): ResourceType | undefined => {
    if (resource.value.startsWith(environmentNamePrefix)) {
      return "environment";
    } else if (isDatabaseName(resource.value)) {
      return "database";
    } else if (resource.value.startsWith(projectNamePrefix)) {
      return "project";
    }
    return undefined;
  });

  const resourcePrefix = computed(() => {
    switch (resourceType.value) {
      case "environment":
        return t("common.environment");
      case "project":
        return t("common.project");
      default:
        return "";
    }
  });

  watchEffect(async () => {
    switch (resourceType.value) {
      case "database":
        if (hasWorkspacePermissionV2("bb.databases.get")) {
          await databaseStore.getOrFetchDatabaseByName(resource.value);
        }
        return;
      case "environment":
        if (hasWorkspacePermissionV2("bb.environments.get")) {
          await environmentV1Store.getOrFetchEnvironmentByName(resource.value);
        }
        return;
      case "project":
        if (hasWorkspacePermissionV2("bb.projects.get")) {
          await projectStore.getOrFetchProjectByName(
            resource.value,
            true /* silent */
          );
        }
        return;
    }
  });

  return {
    resourceType,
    resourcePrefix,
  };
};
