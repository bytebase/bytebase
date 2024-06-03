<template>
  <div class="h-full flex flex-col overflow-y-auto bg-control-bg">
    <nav class="flex-1 flex flex-col overflow-y-hidden">
      <BytebaseLogo class="w-full px-4 shrink-0" />

      <div class="flex-1 overflow-y-auto px-2.5 space-y-1">
        <div v-for="item in itemList" :key="item.name">
          <router-link
            :to="{ path: item.path, name: item.name }"
            class="group flex items-center px-2 py-1.5 leading-normal font-medium rounded-md text-gray-700 outline-item !text-sm"
          >
            <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
            {{ item.title }}
          </router-link>
        </div>
      </div>
    </nav>

    <router-link
      class="flex-shrink-0 flex gap-x-2 justify-start items-center border-t border-block-border px-3 py-1.5 hover:bg-control-bg-hover cursor-pointer"
      :to="{ name: SQL_EDITOR_HOME_MODULE }"
    >
      <ChevronLeftIcon class="w-5 h-5" />
      <span>{{ $t("common.back") }}</span>
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { head } from "lodash-es";
import {
  ChevronLeftIcon,
  GalleryHorizontalEndIcon,
  LayersIcon,
  SquareStackIcon,
} from "lucide-vue-next";
import { computed, h, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter, type RouteRecordRaw } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import { type SidebarItem } from "@/components/CommonSidebar.vue";
import sqlEditorRoutes, {
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_SETTING_INSTANCE_MODULE,
  SQL_EDITOR_SETTING_PROJECT_MODULE,
  SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
  SQL_EDITOR_SETTING_MODULE,
} from "@/router/sqlEditor";
import { useCurrentUserV1 } from "@/store";
import type { WorkspacePermission } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const me = useCurrentUserV1();

const getItemClass = (item: SidebarItem) => {
  if (route.name === item.name) {
    return ["router-link-active", "bg-link-hover"];
  }
  return [];
};

const getFlattenRoutes = (
  routes: RouteRecordRaw[],
  permissions: WorkspacePermission[] = []
): {
  name: string;
  permissions: WorkspacePermission[];
}[] => {
  return routes.reduce(
    (list, workspaceRoute) => {
      const requiredWorkspacePermissionListFunc =
        workspaceRoute.meta?.requiredWorkspacePermissionList;
      let requiredPermissionList = requiredWorkspacePermissionListFunc
        ? requiredWorkspacePermissionListFunc()
        : [];
      if (requiredPermissionList.length === 0) {
        requiredPermissionList = permissions;
      }

      if (workspaceRoute.name && workspaceRoute.name.toString() !== "") {
        list.push({
          name: workspaceRoute.name.toString(),
          permissions: requiredPermissionList,
        });
      }
      if (workspaceRoute.children) {
        list.push(
          ...getFlattenRoutes(workspaceRoute.children, requiredPermissionList)
        );
      }
      return list;
    },
    [] as { name: string; permissions: WorkspacePermission[] }[]
  );
};

const flattenRoutes = computed(() => {
  return getFlattenRoutes(sqlEditorRoutes);
});

const filterSidebarByPermissions = (items: SidebarItem[]): SidebarItem[] => {
  return items
    .filter((item) => {
      const routeConfig = flattenRoutes.value.find(
        (workspaceRoute) => workspaceRoute.name === item.name
      );
      return (routeConfig?.permissions ?? []).every((permission) =>
        hasWorkspacePermissionV2(me.value, permission)
      );
    })
    .map((item) => ({
      ...item,
      expand:
        item.expand ||
        (item.children ?? [])
          .reduce((classList, child) => {
            classList.push(...getItemClass(child));
            return classList;
          }, [] as string[])
          .includes("router-link-active"),
      children: filterSidebarByPermissions(item.children ?? []),
    }));
};

const itemList = computed((): SidebarItem[] => {
  const sidebarList: SidebarItem[] = [
    {
      title: t("common.instances"),
      icon: h(LayersIcon),
      name: SQL_EDITOR_SETTING_INSTANCE_MODULE,
      type: "route",
    },
    {
      title: t("common.projects"),
      icon: h(GalleryHorizontalEndIcon),
      name: SQL_EDITOR_SETTING_PROJECT_MODULE,
      type: "route",
    },
    {
      title: t("common.environments"),
      icon: h(SquareStackIcon),
      name: SQL_EDITOR_SETTING_ENVIRONMENT_MODULE,
      type: "route",
    },
  ];

  return filterSidebarByPermissions(sidebarList);
});

watch(
  () => route.name,
  (name) => {
    if (name === SQL_EDITOR_SETTING_MODULE) {
      const first = head(itemList.value);
      if (first) {
        router.replace({ name: first.name });
      } else {
        router.replace({ name: "error.404" });
      }
    }
  },
  { immediate: true }
);
</script>
