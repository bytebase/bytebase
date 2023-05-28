<template>
  <div class="w-full">
    <div class="w-full flex flex-row justify-between items-center">
      <span>Database groups</span>
      <div class="flex flex-row gap-x-2">
        <NButton>New table group</NButton>
        <NButton @click="state.showCreatingDatabaseGroup = true"
          >New database group</NButton
        >
      </div>
    </div>
    <div class="mt-4">
      <DatabaseGroupTable :database-group-list="databaseGroupList" />
    </div>
  </div>

  <DatabaseGroupPanel
    v-if="state.showCreatingDatabaseGroup"
    :project="project"
    @close="state.showCreatingDatabaseGroup = false"
  />
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { useDBGroupStore } from "@/store";
import { ComposedProject } from "@/types";
import DatabaseGroupTable from "./DatabaseGroupTable.vue";
import DatabaseGroupPanel from "./DatabaseGroupPanel.vue";

interface LocalState {
  showCreatingDatabaseGroup: boolean;
}

const props = defineProps<{
  project: ComposedProject;
}>();

const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  showCreatingDatabaseGroup: false,
});

const databaseGroupList = computed(() => {
  return dbGroupStore.getDBGroupListByProjectName(props.project.name);
});

onMounted(async () => {
  await dbGroupStore.getOrFetchDBGroupListByProjectName(props.project.name);
});
</script>
