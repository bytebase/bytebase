<template>
  <CommonSidebar
    :key="'project'"
    :item-list="projectSidebarItemList"
    :get-item-class="getItemClass"
    :logo-redirect="PROJECT_V1_ROUTE_DETAIL"
    @select="onSelect"
  />
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import CommonSidebar from "@/components/v2/Sidebar/CommonSidebar.vue";
import type { SidebarItem } from "@/components/v2/Sidebar/type";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useCurrentProjectV1 } from "@/store";
import { getProjectName } from "@/store/modules/v1/common";
import { useProjectSidebar } from "./useProjectSidebar";

defineProps<{
  projectId?: string;
  instanceId?: string;
  databaseName?: string;
  changelogId?: string;
}>();

const router = useRouter();
const { record } = useRecentVisit();

const { project } = useCurrentProjectV1();

const { projectSidebarItemList, checkIsActive } = useProjectSidebar(project);

const getItemClass = (item: SidebarItem) => {
  const list = ["outline-item"];
  if (checkIsActive(item)) {
    list.push("router-link-active", "bg-link-hover");
  }
  return list;
};

const onSelect = (item: SidebarItem, e: MouseEvent | undefined) => {
  if (!item.path) {
    return;
  }
  const route = router.resolve({
    name: item.path,
    params: {
      projectId: getProjectName(project.value.name),
    },
  });
  record(route.fullPath);

  if (e?.ctrlKey || e?.metaKey) {
    window.open(route.fullPath, "_blank");
  } else {
    router.push(route);
  }
};
</script>
