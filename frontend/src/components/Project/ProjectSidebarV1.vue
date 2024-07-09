<template>
  <CommonSidebar
    :key="'project'"
    :item-list="projectSidebarItemList"
    :get-item-class="getItemClass"
    @select="onSelect"
  />
</template>

<script setup lang="ts">
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { SidebarItem } from "@/components/CommonSidebar.vue";
import { getProjectName } from "@/store/modules/v1/common";
import { useProjectDatabaseActions } from "../KBar/useDatabaseActions";
import { useCurrentProject } from "./useCurrentProject";
import { useProjectSidebar } from "./useProjectSidebar";

const props = defineProps<{
  projectId?: string;
  instanceId?: string;
  databaseName?: string;
  changeHistoryId?: string;
  issueSlug?: string;
}>();

const { t } = useI18n();
const router = useRouter();

const params = computed(() => {
  return {
    projectId: props.projectId,
    instanceId: props.instanceId,
    databaseName: props.databaseName,
    changeHistoryId: props.changeHistoryId,
    issueSlug: props.issueSlug,
  };
});

const { project } = useCurrentProject(params);

const { projectSidebarItemList, flattenNavigationItems, checkIsActive } =
  useProjectSidebar(project);

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

  if (e?.ctrlKey || e?.metaKey) {
    window.open(route.fullPath, "_blank");
  } else {
    router.replace(route);
  }
};

const navigationKbarActions = computed(() => {
  const actions = flattenNavigationItems.value
    .filter((item) => !item.hide && item.path)
    .map((item) =>
      defineAction({
        id: `bb.navigation.project.${project.value.uid}.${item.path}`,
        name: item.title,
        section: t("kbar.navigation"),
        keywords: [item.title.toLowerCase(), item.path].join(" "),
        perform: () => onSelect(item, undefined),
      })
    );
  return actions;
});
useRegisterActions(navigationKbarActions);

useProjectDatabaseActions(project, 10);
</script>
