<template>
  <nav
    class="flex flex-row justify-between"
    aria-label="Breadcrumb"
    data-label="bb-breadcrumb"
  >
    <div class="flex flex-row grow items-center">
      <div v-for="(item, index) in breadcrumbList" :key="index">
        <div class="flex items-center space-x-2">
          <router-link
            v-if="index == 0"
            to="/"
            class="text-control-light hover:text-control-light-hover"
            active-class="link"
            exact-active-class="link"
          >
            <!-- Heroicon name: solid/home -->
            <heroicons-solid:home class="flex-shrink-0 h-4 w-4" />
            <span class="sr-only">Home</span>
          </router-link>
          <heroicons-solid:chevron-right
            class="ml-2 flex-shrink-0 h-4 w-4 text-control-light"
          />
          <router-link
            v-if="item.path"
            :to="item.path"
            class="text-sm anchor-link max-w-prose truncate"
            active-class="anchor-link"
            exact-active-class="anchor-link"
            >{{ item.name }}</router-link
          >
          <div v-else class="text-sm max-w-prose truncate">
            {{ item.name }}
          </div>
          <span
            v-if="isTenantProject && index == 1"
            class="flex-shrink-0 h-4 w-4"
          >
            <TenantIcon class="ml-1 text-control" />
          </span>
        </div>
      </div>
    </div>

    <div class="tooltip-wrapper">
      <span class="tooltip-left whitespace-nowrap">
        {{ $t("common.show-help") }}
      </span>
      <HelpTriggerIcon v-if="helpName" :id="helpName" :is-guide="true" />
    </div>
  </nav>
</template>

<script lang="ts">
import { useTitle } from "@vueuse/core";
import { computed, ComputedRef, defineComponent, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import HelpTriggerIcon from "@/components/HelpTriggerIcon.vue";
import TenantIcon from "@/components/TenantIcon.vue";
import { useRouterStore, useProjectV1Store, useDatabaseV1Store } from "@/store";
import { RouteMapList } from "@/types";
import { TenantMode } from "@/types/proto/v1/project_service";
import { databaseV1Slug, idFromSlug, projectV1Slug } from "../utils";

interface BreadcrumbItem {
  name: string;
  path?: string;
}

export default defineComponent({
  // eslint-disable-next-line vue/multi-word-component-names
  name: "Breadcrumb",
  components: {
    HelpTriggerIcon,
    TenantIcon,
  },
  setup() {
    const routerStore = useRouterStore();
    const currentRoute = useRouter().currentRoute;
    const { t } = useI18n();
    const projectV1Store = useProjectV1Store();

    const documentTitle = useTitle(null, { observe: true });

    const routeHelpNameMapList = ref<RouteMapList>([]);
    const helpName = computed(
      () =>
        routeHelpNameMapList.value.find(
          (pair) => pair.routeName === currentRoute.value.name
        )?.helpName
    );

    onMounted(async () => {
      const res = await fetch("/help/routeMapList.json");
      routeHelpNameMapList.value = await res.json();
    });

    const isTenantProject: ComputedRef<boolean> = computed(() => {
      const routeSlug = routerStore.routeSlug(currentRoute.value);
      const projectSlug = routeSlug.projectSlug;
      if (projectSlug === undefined) {
        return false;
      }
      const project = projectV1Store.getProjectByUID(
        String(idFromSlug(projectSlug))
      );
      return project.tenantMode == TenantMode.TENANT_MODE_ENABLED;
    });

    const breadcrumbList = computed(() => {
      const route = currentRoute.value;
      const routeSlug = routerStore.routeSlug(currentRoute.value);
      const environmentSlug = routeSlug.environmentSlug;
      const projectSlug = routeSlug.projectSlug;
      const projectWebhookSlug = routeSlug.projectWebhookSlug;
      const instanceSlug = routeSlug.instanceSlug;
      const databaseSlug = routeSlug.databaseSlug;
      const tableName = routeSlug.tableName;
      const vcsSlug = routeSlug.vcsSlug;
      const sqlReviewPolicySlug = routeSlug.sqlReviewPolicySlug;
      const ssoName = routeSlug.ssoName;

      const changelistName = routeSlug.changelistName;
      const databaseGroupName = routeSlug.databaseGroupName;
      const schemaGroupName = routeSlug.schemaGroupName;

      const list: Array<BreadcrumbItem> = [];
      if (environmentSlug) {
        list.push({
          name: t("common.environments"),
          path: "/environment",
        });
      } else if (projectSlug) {
        list.push({
          name: t("common.projects"),
          path: "/project",
        });

        const project = projectV1Store.getProjectByUID(
          String(idFromSlug(projectSlug))
        );

        if (projectWebhookSlug) {
          list.push({
            name: `${project.title}`,
            path: `/project/${projectSlug}`,
          });
        } else if (databaseGroupName) {
          list.push(
            {
              name: project.title,
              path: `/project/${projectV1Slug(project)}`,
            },
            {
              name: t("common.database-groups"),
              path: `/project/${projectSlug}#database-groups`,
            }
          );

          if (schemaGroupName) {
            list.push(
              {
                name: databaseGroupName,
                path: `/project/${projectSlug}/database-groups/${databaseGroupName}`,
              },
              {
                name: `Tables - ${schemaGroupName}`,
              }
            );
          } else {
            list.push({
              name: databaseGroupName,
            });
          }
        } else if (changelistName) {
          list.push(
            {
              name: project.title,
              path: `/project/${projectV1Slug(project)}`,
            },
            {
              name: t("changelist.self"),
              path: `/project/${projectSlug}#changelists`,
            }
          );
        } else if (route.name === "workspace.branch.detail") {
          list.push(
            {
              name: project.title,
              path: `/project/${projectV1Slug(project)}`,
            },
            {
              name: t("common.branches"),
              path: `/project/${projectV1Slug(project)}#branches`,
            }
          );
        }
      } else if (instanceSlug) {
        list.push({
          name: t("common.instances"),
          path: "/instance",
        });
      } else if (databaseSlug) {
        const database = useDatabaseV1Store().getDatabaseByUID(
          String(idFromSlug(databaseSlug))
        );

        list.push(
          {
            name: t("common.projects"),
            path: "/project",
          },
          {
            name: `${database.projectEntity.title}`,
            path: `/project/${projectV1Slug(database.projectEntity)}`,
          },
          {
            name: t("common.databases"),
            path: "/db",
          }
        );

        if (tableName) {
          list.push({
            name: database.databaseName,
            path: `/db/${databaseSlug}`,
          });
        }
      } else if (vcsSlug) {
        list.push({
          name: t("common.gitops"),
          path: "/setting/gitops",
        });
      } else if (sqlReviewPolicySlug) {
        list.push({
          name: t("sql-review.title"),
          path: "/setting/sql-review",
        });
      } else if (ssoName) {
        if (route.name !== "setting.workspace.sso.create") {
          list.push({
            name: t("settings.sidebar.sso"),
            path: "/setting/sso",
          });
        }
      }
      if (route.name === "workspace.database.history.detail") {
        const parent = `instances/${route.params.instance}/databases/${route.params.database}`;
        const database = useDatabaseV1Store().getDatabaseByName(parent);

        list.push(
          {
            name: t("common.projects"),
            path: "/project",
          },
          {
            name: `${database.projectEntity.title}`,
            path: `/project/${projectV1Slug(database.projectEntity)}`,
          },
          {
            name: t("common.databases"),
            path: "/db",
          },
          {
            name: database.databaseName,
            path: `/db/${databaseV1Slug(database)}`,
          },
          {
            name: t("common.change"),
            path: `/db/${databaseV1Slug(database)}#change-history`,
          }
        );
      } else if ((route.name ?? "")?.toString().startsWith("setting.")) {
        list.push({
          name: t("common.settings"),
        });
      }

      const {
        title: routeTitle,
        overrideTitle,
        overrideBreadcrumb,
      } = route.meta;

      // Dynamic title priorities
      // 1. documentTitle - if (overrideTitle === true)
      // 2. routeTitle - if (routeTitle !== undefined)
      // 3. nothing - otherwise
      const title = overrideTitle
        ? documentTitle.value ?? ""
        : routeTitle?.(route) ?? "";
      if (title) {
        if (overrideBreadcrumb && overrideBreadcrumb(route)) {
          list.length = 0; // empty the array
        }
        list.push({
          name: title,
          // Set empty path for the current route to make the link not clickable.
          // We do this because clicking the current route path won't trigger reload and would
          // confuse user since UI won't change while we may have cleared all query parameters.
          path: "",
        });
      }

      return list;
    });

    return {
      isTenantProject,
      breadcrumbList,
      currentRoute,
      helpName,
    };
  },
});
</script>
