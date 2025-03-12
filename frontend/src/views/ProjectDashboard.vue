<template>
  <div class="flex flex-col space-y-4">
    <div class="px-4 flex items-center space-x-2">
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
      :search="state.searchText"
      :footer-class="'mx-4'"
      :bordered="false"
      :include-default="false"
    />
  </div>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <ProjectCreatePanel @dismiss="state.showCreateDrawer = false" />
  </Drawer>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { onMounted, reactive } from "vue";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import { SearchBox, PagedProjectTable } from "@/components/v2";
import { Drawer } from "@/components/v2";
import { useUIStateStore } from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  searchText: string;
  showCreateDrawer: boolean;
}

const state = reactive<LocalState>({
  searchText: "",
  showCreateDrawer: false,
});

onMounted(() => {
  const uiStateStore = useUIStateStore();
  if (!uiStateStore.getIntroStateByKey("project.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "project.visit",
      newState: true,
    });
  }
});
</script>
