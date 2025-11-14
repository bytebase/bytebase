import { computed, h, type Ref, watchEffect } from "vue";
import { EnvironmentV1Name, RichDatabaseName } from "@/components/v2";
import { ProjectNameCell } from "@/components/v2/Model/cells";
import { t } from "@/plugins/i18n";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
  useUserStore,
} from "@/store";
import {
  environmentNamePrefix,
  isDatabaseName,
  projectNamePrefix,
  userNamePrefix,
} from "@/store/modules/v1/common";
import {
  formatEnvironmentName,
  isValidDatabaseName,
  isValidEnvironmentName,
  isValidProjectName,
} from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

type ResourceType =
  | "environment"
  | "project"
  | "database"
  | "workspace"
  | "user";

export const useResourceByName = ({
  resource,
  link,
}: {
  resource: Ref<string>;
  link?: boolean;
}) => {
  const environmentV1Store = useEnvironmentV1Store();
  const databaseStore = useDatabaseV1Store();
  const projectStore = useProjectV1Store();
  const userStore = useUserStore();

  const resourceType = computed((): ResourceType | undefined => {
    if (resource.value === "workspace") {
      return "workspace";
    }
    if (resource.value.startsWith(environmentNamePrefix)) {
      return "environment";
    } else if (isDatabaseName(resource.value)) {
      return "database";
    } else if (resource.value.startsWith(projectNamePrefix)) {
      return "project";
    } else if (resource.value.startsWith(userNamePrefix)) {
      return "user";
    }
    return undefined;
  });

  const resourcePrefix = computed(() => {
    switch (resourceType.value) {
      case "environment":
        return t("common.environment");
      case "project":
        return t("common.project");
      case "database":
        return t("common.database");
      case "user":
        return t("common.user");
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
        if (hasWorkspacePermissionV2("bb.settings.get")) {
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
      case "user":
        await userStore.getOrFetchUserByIdentifier(resource.value, true);
        return;
    }
  });

  const resourceComponent = computed(() => {
    switch (resourceType.value) {
      case "environment": {
        const environment = environmentV1Store.getEnvironmentByName(
          resource.value
        );
        if (!isValidEnvironmentName(formatEnvironmentName(environment.id))) {
          return h("div", resource.value);
        }
        return h(EnvironmentV1Name, { environment, link });
      }
      case "database": {
        const database = databaseStore.getDatabaseByName(resource.value);
        if (!isValidDatabaseName(database.name)) {
          return h("div", resource.value);
        }
        return h(RichDatabaseName, {
          database,
          showArrow: false,
          showInstance: false,
        });
      }
      case "project": {
        const project = projectStore.getProjectByName(resource.value);
        if (!isValidProjectName(project.name)) {
          return h("div", resource.value);
        }
        return h(ProjectNameCell, {
          project,
          mode: "ALL_SHORT",
          link,
        });
      }
      case "user": {
        const user = userStore.getUserByIdentifier(resource.value);
        if (!user) {
          return h("div", resource.value);
        }
        return h(
          "a",
          {
            class: "flex items-center gap-x-2 normal-link underline",
            href: `/${user.name}`,
          },
          `${user.title} (${user.email})`
        );
      }
      default:
        return h("div", resource.value);
    }
  });

  return {
    resourceType,
    resourcePrefix,
    resourceComponent,
  };
};
