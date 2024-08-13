<template>
  <div
    class="flex-1 flex flex-col gap-2 relative overflow-auto"
    style="width: calc(100vw - 8rem); max-width: 50rem"
  >
    <div
      class="flex flex-col items-start gap-2 sm:flex-row sm:justify-between sm:items-center"
    >
      <div class="flex justify-start items-center">
        <NButton @click="handleClickTransfer">
          <template #icon>
            <ChevronsDownIcon class="w-4 h-4" />
          </template>
          {{ $t("quick-action.transfer-in-db") }}
        </NButton>
      </div>
    </div>

    <div class="relative">
      <DatabaseV1Table
        v-if="ready"
        mode="PROJECT_SHORT"
        :database-list="databaseList"
        :custom-click="true"
        :show-selection="false"
        :row-clickable="false"
      />
    </div>
    <MaskSpinner v-if="!ready" />

    <Drawer v-model:show="showTransfer">
      <TransferDatabaseForm
        :project-name="project.name"
        :on-success="handleTransferSuccess"
        @dismiss="showTransfer = false"
      />
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { ChevronsDownIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { ref } from "vue";
import TransferDatabaseForm from "@/components/TransferDatabaseForm.vue";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { Drawer } from "@/components/v2";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { Project } from "@/types/proto/v1/project_service";

const props = defineProps<{
  project: Project;
}>();

const { databaseList, ready } = useDatabaseV1List(props.project.name);
const showTransfer = ref(false);

const handleClickTransfer = () => {
  showTransfer.value = true;
};

const handleTransferSuccess = () => {
  showTransfer.value = false;
};
</script>
