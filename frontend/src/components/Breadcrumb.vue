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
          <button
            v-if="allowBookmark && index == breadcrumbList.length - 1"
            class="relative focus:outline-none"
            type="button"
            @click.prevent="toggleBookmark"
          >
            <heroicons-solid:star
              v-if="isBookmarked"
              class="h-5 w-5 text-yellow-400 hover:text-yellow-600"
            />
            <heroicons-solid:star
              v-else
              class="h-5 w-5 text-control-light hover:text-control-light-hover"
            />
          </button>
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
import {
  useRouterStore,
  useBookmarkV1Store,
  useProjectV1Store,
  useDatabaseV1Store,
} from "@/store";
import { Bookmark } from "@/types/proto/v1/bookmark_service";
import { RouteMapList } from "../types";
import { databaseV1Slug, idFromSlug } from "../utils";

interface BreadcrumbItem {
  name: string;
  path?: string;
}

export default defineComponent({
  // eslint-disable-next-line vue/multi-word-component-names
  name: "Breadcrumb",
  components: {
    HelpTriggerIcon,
  },
  setup() {
    const routerStore = useRouterStore();
    const currentRoute = useRouter().currentRoute;
    const { t } = useI18n();
    const bookmarkV1Store = useBookmarkV1Store();

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

    const bookmark: ComputedRef<Bookmark | undefined> = computed(() =>
      bookmarkV1Store.findBookmarkByLink(currentRoute.value.path)
    );

    const isBookmarked: ComputedRef<boolean> = computed(() => !!bookmark.value);

    const allowBookmark = computed(() => currentRoute.value.meta.allowBookmark);

    const breadcrumbList = computed(() => {
      const route = currentRoute.value;
      const routeSlug = routerStore.routeSlug(currentRoute.value);
      const environmentSlug = routeSlug.environmentSlug;
      const projectSlug = routeSlug.projectSlug;
      const projectWebhookSlug = routeSlug.projectWebhookSlug;
      const instanceSlug = routeSlug.instanceSlug;
      const databaseSlug = routeSlug.databaseSlug;
      const tableName = routeSlug.tableName;
      const dataSourceSlug = routeSlug.dataSourceSlug;
      const vcsSlug = routeSlug.vcsSlug;
      const sqlReviewPolicySlug = routeSlug.sqlReviewPolicySlug;
      const ssoName = routeSlug.ssoName;

      const projectName = routeSlug.projectName;
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

        if (projectWebhookSlug) {
          const project = projectV1Store.getProjectByUID(
            String(idFromSlug(projectSlug))
          );
          list.push({
            name: `${project.title}`,
            path: `/project/${projectSlug}`,
          });
        }
      } else if (instanceSlug) {
        list.push({
          name: t("common.instances"),
          path: "/instance",
        });
      } else if (databaseSlug) {
        list.push({
          name: t("common.databases"),
          path: "/db",
        });

        if (tableName || dataSourceSlug) {
          const database = useDatabaseV1Store().getDatabaseByUID(
            String(idFromSlug(databaseSlug))
          );
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
      } else if (schemaGroupName) {
        if (projectName && databaseGroupName) {
          list.push(
            {
              name: "Databases",
            },
            {
              name: databaseGroupName,
              path: `/projects/${projectName}/database-groups/${databaseGroupName}`,
            },
            {
              name: `Tables - ${schemaGroupName}`,
            }
          );
        }
      }
      if (route.name === "workspace.database.history.detail") {
        const parent = `instances/${route.params.instance}/databases/${route.params.database}`;
        const database = useDatabaseV1Store().getDatabaseByName(parent);
        list.push({
          name: database.databaseName,
          path: `/db/${databaseV1Slug(database)}`,
        });
        list.push({
          name: t("common.change"),
          path: `/db/${databaseV1Slug(database)}#change-history`,
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

    const toggleBookmark = () => {
      if (bookmark.value) {
        bookmarkV1Store.deleteBookmark(bookmark.value.name);
      } else {
        bookmarkV1Store.createBookmark({
          title: breadcrumbList.value[breadcrumbList.value.length - 1].name,
          link: currentRoute.value.path,
        });
      }
    };

    return {
      allowBookmark,
      bookmark,
      isBookmarked,
      breadcrumbList,
      toggleBookmark,
      currentRoute,
      helpName,
    };
  },
});
</script>
