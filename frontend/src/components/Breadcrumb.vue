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

    <HelpTriggerIcon
      v-if="currentRoute.name in routeHelpNameMap"
      :name="routeHelpNameMap[currentRoute.name]"
      :is-guide="true"
    />
  </nav>
</template>

<script lang="ts">
import { computed, ComputedRef, defineComponent } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Bookmark, UNKNOWN_ID, BookmarkCreate } from "../types";
import { idFromSlug } from "../utils";
import {
  useCurrentUser,
  useRouterStore,
  useUIStateStore,
  useBookmarkStore,
  useDatabaseStore,
  useProjectStore,
} from "@/store";
import HelpTriggerIcon from "@/components/HelpTriggerIcon.vue";
import { routeHelpNameMap } from "../../public/help/config";

interface BreadcrumbItem {
  name: string;
  path?: string;
}

export default defineComponent({
  name: "Breadcrumb",
  components: {
    HelpTriggerIcon,
  },
  setup() {
    const routerStore = useRouterStore();
    const currentRoute = useRouter().currentRoute;
    const { t } = useI18n();
    const bookmarkStore = useBookmarkStore();

    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();

    const bookmark: ComputedRef<Bookmark> = computed(() =>
      bookmarkStore.bookmarkByUserAndLink(
        currentUser.value.id,
        currentRoute.value.path
      )
    );

    const isBookmarked: ComputedRef<boolean> = computed(
      () => bookmark.value.id != UNKNOWN_ID
    );

    const allowBookmark = computed(() => currentRoute.value.meta.allowBookmark);

    const breadcrumbList = computed(() => {
      const routeSlug = routerStore.routeSlug(currentRoute.value);
      const environmentSlug = routeSlug.environmentSlug;
      const projectSlug = routeSlug.projectSlug;
      const projectWebhookSlug = routeSlug.projectWebhookSlug;
      const instanceSlug = routeSlug.instanceSlug;
      const databaseSlug = routeSlug.databaseSlug;
      const tableName = routeSlug.tableName;
      const dataSourceSlug = routeSlug.dataSourceSlug;
      const migrationHistory = routeSlug.migrationHistorySlug;
      const versionControlSlug = routeSlug.vcsSlug;
      const sqlReviewPolicySlug = routeSlug.sqlReviewPolicySlug;

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
          const project = projectStore.getProjectById(idFromSlug(projectSlug));
          list.push({
            name: `${project.name}`,
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

        if (tableName || dataSourceSlug || migrationHistory) {
          const database = useDatabaseStore().getDatabaseById(
            idFromSlug(databaseSlug)
          );
          list.push({
            name: database.name,
            path: `/db/${databaseSlug}`,
          });
          if (migrationHistory) {
            list.push({
              name: t("common.migration"),
              path: `/db/${databaseSlug}#migration-history`,
            });
          }
        }
      } else if (versionControlSlug) {
        list.push({
          name: t("common.version-control"),
          path: "/setting/version-control",
        });
      } else if (sqlReviewPolicySlug) {
        list.push({
          name: t("sql-review.title"),
          path: "/setting/sql-review",
        });
      }

      if (currentRoute.value.meta.title) {
        list.push({
          name: currentRoute.value.meta.title(currentRoute.value),
          // Set empty path for the current route to make the link not clickable.
          // We do this because clicking the current route path won't trigger reload and would
          // confuse user since UI won't change while we may have cleared all query parameters.
          path: "",
        });
      }

      return list;
    });

    const toggleBookmark = () => {
      if (isBookmarked.value) {
        bookmarkStore.deleteBookmark(bookmark.value);
      } else {
        const newBookmark: BookmarkCreate = {
          name: breadcrumbList.value[breadcrumbList.value.length - 1].name,
          link: currentRoute.value.path,
        };
        bookmarkStore.createBookmark(newBookmark).then(() => {
          useUIStateStore().saveIntroStateByKey({
            key: "bookmark.create",
            newState: true,
          });
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
      routeHelpNameMap,
    };
  },
});
</script>
