<template>
  <BBModal
    :close-on-esc="true"
    :mask-closable="true"
    :trap-focus="false"
    :title="$t('project.select')"
    class="w-[48rem] max-w-full h-128 max-h-full"
    @close="$emit('dismiss')"
  >
    <div class="space-y-2 my-4">
      <div class="w-full sticky top-0 mb-4">
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
      <NTabs v-model:value="state.selectedTab" type="line">
        <NTabPane
          v-for="tab in tabList"
          :name="tab.id"
          :key="tab.id"
          :tab="tab.title"
        >
          <ProjectV1Table
            :project-list="tab.list"
            :current-project="currentProject"
            class="border"
            @click="$emit('dismiss')"
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
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import { SearchBox, ProjectV1Table } from "@/components/v2";
import { Drawer } from "@/components/v2";
import { useProjectV1ListByCurrentUser } from "@/store";
import {
  DEFAULT_PROJECT_ID,
  UNKNOWN_PROJECT_NAME,
  EMPTY_PROJECT_NAME,
  ComposedProject,
} from "@/types";
import { filterProjectV1ListByKeyword } from "@/utils";

interface LocalState {
  searchText: string;
  showCreateDrawer: boolean;
  selectedTab: "recent" | "all";
}

const props = defineProps<{
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
const { projectList } = useProjectV1ListByCurrentUser();
const { recentViewProjects } = useRecentProjects();

onMounted(() => {
  state.selectedTab = recentViewProjects.value.length < 1 ? "all" : "recent";
});

const getFilteredProjectList = (
  projectList: ComposedProject[]
): ComposedProject[] => {
  const list = projectList.filter(
    (project) => project.uid !== String(DEFAULT_PROJECT_ID)
  );
  return filterProjectV1ListByKeyword(list, state.searchText);
};

const tabList = computed(() => [
  {
    title: t("common.recent"),
    id: "recent",
    list: getFilteredProjectList(recentViewProjects.value),
  },
  {
    title: t("common.all"),
    id: "all",
    list: getFilteredProjectList(projectList.value),
  },
]);

const currentProject = computed(() => {
  if (
    props.project?.name === UNKNOWN_PROJECT_NAME ||
    props.project?.name === EMPTY_PROJECT_NAME
  ) {
    return undefined;
  }
  return props.project;
});

const onCreate = () => {
  state.showCreateDrawer = false;
  emit("dismiss");
};
</script>
