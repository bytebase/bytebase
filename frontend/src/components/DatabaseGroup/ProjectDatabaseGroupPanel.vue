<template>
  <div class="w-full">
    <div class="w-full flex flex-row justify-between items-center">
      <span>Database groups</span>
      <div class="flex flex-row gap-x-2">
        <NButton>New table group</NButton>
        <NButton>New database group</NButton>
      </div>
    </div>
    <div class="mt-4">
      <DatabaseGroupTable :database-group-list="state.databaseGroupList" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { onMounted, reactive } from "vue";
import DatabaseGroupTable from "./DatabaseGroupTable.vue";

interface LocalState {
  databaseGroupList: DatabaseGroup[];
}

const state = reactive<LocalState>({
  databaseGroupList: [],
});

onMounted(()=>{

})
</script>
