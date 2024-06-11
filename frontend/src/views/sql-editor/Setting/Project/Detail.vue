<template>
  <div
    class="flex-1 flex flex-col gap-2 relative overflow-auto"
    style="width: calc(100vw - 8rem); max-width: 50rem"
  >
    <div
      class="flex flex-col items-start gap-2 sm:flex-row sm:justify-between sm:items-center"
    >
      <div class="flex justify-start items-center">
        <NButton>
          <template #icon>
            <ChevronsDownIcon class="w-4 h-4" />
          </template>
          {{ $t("quick-action.transfer-in-db") }}
        </NButton>
      </div>
    </div>

    <DatabaseV1Table
      v-if="ready"
      mode="PROJECT_SHORT"
      :database-list="databaseList"
      :custom-click="true"
      :show-selection="false"
      :show-sql-editor-button="false"
      :row-clickable="false"
    />
    <MaskSpinner v-if="!ready || refreshing" />
  </div>
</template>

<script setup lang="ts">
import { ChevronsDownIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { ref, watch } from "vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table";
import { databaseServiceClient } from "@/grpcweb";
import { DEFAULT_DATABASE_PAGE_SIZE, batchComposeDatabase } from "@/store";
import type { ComposedDatabase, ComposedProject } from "@/types";

const props = defineProps<{
  project: ComposedProject;
}>();

const ready = ref(false);
const refreshing = ref(false);
const databaseList = ref<ComposedDatabase[]>([]);

const fetchDatabaseList = async (force: boolean) => {
  refreshing.value = true;
  if (force) {
    ready.value = false;
  }
  const response = await databaseServiceClient.listDatabases({
    parent: "instances/-",
    filter: `project == "${props.project.name}"`,
    pageSize: DEFAULT_DATABASE_PAGE_SIZE,
  });

  const list = await batchComposeDatabase(response.databases);

  databaseList.value = list;

  refreshing.value = false;
  ready.value = true;
};

watch(
  () => props.project.name,
  () => fetchDatabaseList(/* force */ true),
  { immediate: true }
);
</script>
