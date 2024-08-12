<template>
  <BBModal
    :close-on-esc="true"
    :mask-closable="true"
    :trap-focus="false"
    :show="show"
    :title="$t('project.select')"
    class="w-[48rem] max-w-full h-128 max-h-full"
    @close="$emit('dismiss')"
  >
    <div class="h-full overflow-y-auto relative">
      <div v-if="currentProject" class="mb-2">
        <NButton text @click="gotoWorkspace">
          <template #icon>
            <ChevronLeftIcon class="w-4" />
          </template>
          {{ $t("common.back-to-workspace") }}
        </NButton>
      </div>
      <div class="w-full bg-white sticky top-0 z-50">
        <div class="flex items-center justify-between space-x-2">
          <SearchBox
            v-model:value="state.searchText"
            :placeholder="$t('common.filter-by-name')"
            :autofocus="false"
            style="max-width: 100%"
          />
          <NButton @click="state.showCreateDrawer = true">
            {{ $t("quick-action.new-project") }}
          </NButton>
        </div>
      </div>
      <NTabs
        :value="actualSelectedTab"
        type="line"
        @update:value="state.selectedTab = $event"
      >
        <NTabPane
          v-for="tab in tabList"
          :key="tab.id"
          :name="tab.id"
          :tab="tab.title"
          :disabled="
            tab.id === 'recent' &&
            state.searchText.trim().length > 0 &&
            filteredRecentProjectList.length === 0
          "
        >
          <ProjectV1Table
            :project-list="tab.list"
            :current-project="currentProject"
            :pagination="false"
            :keyword="state.searchText"
            @row-click="onProjectSelect"
          />
        </NTabPane>
      </NTabs>
    </div>
  </BBModal>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <ProjectCreatePanel @dismiss="onCreate" />
  </Drawer>
</template>

<script lang="ts" setup>
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBModal } from "@/bbkit";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import { SearchBox, ProjectV1Table } from "@/components/v2";
import { Drawer } from "@/components/v2";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_MY_ISSUES } from "@/router/dashboard/workspaceRoutes";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useProjectV1List } from "@/store";
import { getProjectName } from "@/store/modules/v1/common";
import type { ComposedProject } from "@/types";
import { isValidProjectName, DEFAULT_PROJECT_NAME } from "@/types";
import { filterProjectV1ListByKeyword } from "@/utils";
import ProjectCreatePanel from "./ProjectCreatePanel.vue";

interface LocalState {
  searchText: string;
  showCreateDrawer: boolean;
  selectedTab: "recent" | "all";
}

const props = defineProps<{
  show: boolean;
  project?: ComposedProject;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  searchText: "",
  showCreateDrawer: false,
  selectedTab: "all",
});
const { projectList } = useProjectV1List();
const { recentViewProjects } = useRecentProjects();
const router = useRouter();
const { record } = useRecentVisit();

onMounted(() => {
  state.selectedTab = recentViewProjects.value.length < 1 ? "all" : "recent";
});

const getFilteredProjectList = (
  projectList: ComposedProject[]
): ComposedProject[] => {
  const list = projectList.filter(
    (project) => project.name !== DEFAULT_PROJECT_NAME
  );
  return filterProjectV1ListByKeyword(list, state.searchText);
};

const filteredRecentProjectList = computed(() => {
  return getFilteredProjectList(recentViewProjects.value);
});
const filteredAllProjectList = computed(() => {
  return getFilteredProjectList(projectList.value);
});

const tabList = computed(() => [
  {
    title: t("common.recent"),
    id: "recent",
    list: filteredRecentProjectList.value,
  },
  {
    title: t("common.all"),
    id: "all",
    list: filteredAllProjectList.value,
  },
]);

const actualSelectedTab = computed((): LocalState["selectedTab"] => {
  if (
    state.searchText.trim().length > 0 &&
    filteredRecentProjectList.value.length === 0
  ) {
    // Force to view 'ALL' tab when search by keyword but "Recent" is empty.
    return "all";
  }
  return state.selectedTab;
});

const currentProject = computed(() => {
  if (!isValidProjectName(props.project?.name)) {
    return undefined;
  }
  return props.project;
});

const onCreate = () => {
  state.showCreateDrawer = false;
  emit("dismiss");
};

const onProjectSelect = (project: ComposedProject) => {
  const route = router.resolve({
    name: PROJECT_V1_ROUTE_DETAIL,
    params: {
      projectId: getProjectName(project.name),
    },
  });
  record(route.fullPath);
  emit("dismiss");
};

const gotoWorkspace = () => {
  router.push({ name: WORKSPACE_ROUTE_MY_ISSUES });
  emit("dismiss");
};
</script>
