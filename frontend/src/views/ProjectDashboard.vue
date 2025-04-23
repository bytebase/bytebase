<template>
  <div class="flex flex-col space-y-4 px-4">
    <div class="flex items-center space-x-2">
      <SearchBox
        v-model:value="state.searchText"
        style="max-width: 100%"
        :placeholder="$t('common.filter-by-name')"
      />
      <NButton
        v-if="hasWorkspacePermissionV2('bb.projects.create')"
        type="primary"
        @click="state.showCreateDrawer = true"
      >
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        {{ $t("quick-action.new-project") }}
      </NButton>
    </div>
    <PagedProjectTable
      session-key="bb.project-table"
      :filter="{
        query: state.searchText,
        excludeDefault: true,
      }"
      bordered
      :prevent-default="!!onRowClick"
      @row-click="onRowClick"
    />
  </div>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <ProjectCreatePanel
      :on-created="handleCreated"
      @dismiss="state.showCreateDrawer = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import { SearchBox, PagedProjectTable } from "@/components/v2";
import { Drawer } from "@/components/v2";
import { useUIStateStore } from "@/store";
import type { ComposedProject } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  searchText: string;
  showCreateDrawer: boolean;
}

const props = defineProps<{
  onRowClick?: (project: ComposedProject) => void;
}>();

const state = reactive<LocalState>({
  searchText: "",
  showCreateDrawer: false,
});
const router = useRouter();

onMounted(() => {
  const uiStateStore = useUIStateStore();
  if (!uiStateStore.getIntroStateByKey("project.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "project.visit",
      newState: true,
    });
  }
});

const handleCreated = async (project: ComposedProject) => {
  if (props.onRowClick) {
    return props.onRowClick(project);
  }
  const url = {
    path: `/${project.name}`,
  };
  router.push(url);
  state.showCreateDrawer = false;
};
</script>
