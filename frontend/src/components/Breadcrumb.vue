<template>
  <nav class="flex" aria-label="Breadcrumb">
    <div v-for="(item, index) in breadcrumbList" :key="index">
      <div class="flex items-center space-x-2">
        <router-link
          v-if="index == 0"
          to="/"
          class="text-control-light hover:text-control-light-hover"
          active-class="link"
          exact-active-class="link"
        >
          <!-- Heroicon name: home -->
          <svg
            class="flex-shrink-0 h-4 w-4"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            aria-hidden="true"
          >
            <path
              d="M10.707 2.293a1 1 0 00-1.414 0l-7 7a1 1 0 001.414 1.414L4 10.414V17a1 1 0 001 1h2a1 1 0 001-1v-2a1 1 0 011-1h2a1 1 0 011 1v2a1 1 0 001 1h2a1 1 0 001-1v-6.586l.293.293a1 1 0 001.414-1.414l-7-7z"
            />
          </svg>
          <span class="sr-only">Home</span>
        </router-link>
        <svg
          class="ml-2 flex-shrink-0 h-4 w-4 text-control-light"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            fill-rule="evenodd"
            d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
            clip-rule="evenodd"
          />
        </svg>
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
          <svg
            v-if="isBookmarked"
            class="h-5 w-5 text-yellow-400 hover:text-yellow-600"
            x-description="Heroicon name: star"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"
            ></path>
          </svg>
          <svg
            v-else
            class="h-5 w-5 text-control-light hover:text-control-light-hover"
            x-description="Heroicon name: star"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"
            ></path>
          </svg>
        </button>
      </div>
    </div>
  </nav>
</template>

<script lang="ts">
import { computed, ComputedRef } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import {
  RouterSlug,
  Bookmark,
  UNKNOWN_ID,
  Principal,
  BookmarkCreate,
} from "../types";
import { idFromSlug } from "../utils";

interface BreadcrumbItem {
  name: string;
  path?: string;
}

export default {
  name: "Breadcrumb",
  components: {},
  setup() {
    const store = useStore();
    const currentRoute = useRouter().currentRoute;

    const currentUser: ComputedRef<Principal> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const bookmark: ComputedRef<Bookmark> = computed(() =>
      store.getters["bookmark/bookmarkByUserAndLink"](
        currentUser.value.id,
        currentRoute.value.path
      )
    );

    const isBookmarked: ComputedRef<boolean> = computed(
      () => bookmark.value.id != UNKNOWN_ID
    );

    const allowBookmark = computed(() => currentRoute.value.meta.allowBookmark);

    const breadcrumbList = computed(() => {
      const routeSlug: RouterSlug = store.getters["router/routeSlug"](
        currentRoute.value
      );
      const environmentSlug = routeSlug.environmentSlug;
      const projectSlug = routeSlug.projectSlug;
      const projectWebhookSlug = routeSlug.projectWebhookSlug;
      const instanceSlug = routeSlug.instanceSlug;
      const databaseSlug = routeSlug.databaseSlug;
      const tableName = routeSlug.tableName;
      const dataSourceSlug = routeSlug.dataSourceSlug;
      const migrationHistory = routeSlug.migrationHistorySlug;
      const versionControlSlug = routeSlug.vcsSlug;

      const list: Array<BreadcrumbItem> = [];
      if (environmentSlug) {
        list.push({
          name: "Environment",
          path: "/environment",
        });
      } else if (projectSlug) {
        list.push({
          name: "Project",
          path: "/project",
        });

        if (projectWebhookSlug) {
          const project = store.getters["project/projectById"](
            idFromSlug(projectSlug)
          );
          list.push({
            name: `${project.name}`,
            path: `/project/${projectSlug}`,
          });
        }
      } else if (instanceSlug) {
        list.push({
          name: "Instance",
          path: "/instance",
        });
      } else if (databaseSlug) {
        list.push({
          name: "Database",
          path: "/db",
        });

        if (tableName || dataSourceSlug || migrationHistory) {
          const database = store.getters["database/databaseById"](
            idFromSlug(databaseSlug)
          );
          list.push({
            name: database.name,
            path: `/db/${databaseSlug}`,
          });
          if (migrationHistory) {
            list.push({
              name: "Migration",
              path: `/db/${databaseSlug}#migration-history`,
            });
          }
        }
      } else if (versionControlSlug) {
        list.push({
          name: "Version Control",
          path: "/setting/version-control",
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
        store.dispatch("bookmark/deleteBookmark", bookmark.value);
      } else {
        const newBookmark: BookmarkCreate = {
          name: breadcrumbList.value[breadcrumbList.value.length - 1].name,
          link: currentRoute.value.path,
        };
        store.dispatch("bookmark/createBookmark", newBookmark).then(() => {
          store.dispatch("uistate/saveIntroStateByKey", {
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
    };
  },
};
</script>
