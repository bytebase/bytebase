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
            style="flex: 1 1 0%"
          />
          <NButton @click="state.showCreateDrawer = true">
            {{ $t("quick-action.new-project") }}
          </NButton>
        </div>
      </div>
      <ProjectV1Table
        :project-list="filteredProjectList"
        class="border"
        @click="$emit('dismiss')"
      />
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
import { computed, reactive } from "vue";
import { SearchBox, ProjectV1Table } from "@/components/v2";
import { Drawer } from "@/components/v2";
import { useProjectV1ListByCurrentUser } from "@/store";
import { DEFAULT_PROJECT_ID } from "@/types";
import { filterProjectV1ListByKeyword } from "@/utils";

interface LocalState {
  searchText: string;
  showCreateDrawer: boolean;
}

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const state = reactive<LocalState>({
  searchText: "",
  showCreateDrawer: false,
});
const { projectList } = useProjectV1ListByCurrentUser();

const filteredProjectList = computed(() => {
  const list = projectList.value.filter(
    (project) => project.uid !== String(DEFAULT_PROJECT_ID)
  );
  return filterProjectV1ListByKeyword(list, state.searchText);
});

const onCreate = () => {
  state.showCreateDrawer = false;
  emit("dismiss");
};
</script>
