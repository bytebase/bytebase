<template>
  <NPopover trigger="hover" placement="bottom">
    <template #trigger>
      <NButton size="tiny">
        <template #icon>
          <EllipsisVerticalIcon />
        </template>
      </NButton>
    </template>

    <div class="flex flex-col items-start justify-start">
      <template v-if="shouldShowInstanceRoleSelect">
        <InstanceRoleSelect />
        <NDivider class="!my-2" />
      </template>
      <template v-if="shouldShowTransactionModeToggle">
        <TransactionModeToggle />
        <NDivider class="!my-2" />
      </template>
      <FormatOnSaveCheckbox v-model:value="formatOnSave" :language="language" />
    </div>
  </NPopover>
</template>

<script setup lang="tsx">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { NPopover, NButton, NDivider } from "naive-ui";
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { useCurrentProjectV1 } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import {
  useInstanceV1EditorLanguage,
  instanceV1SupportsTransactionMode,
} from "@/utils";
import FormatOnSaveCheckbox from "./FormatOnSaveCheckbox.vue";
import InstanceRoleSelect from "./InstanceRoleSelect.vue";
import TransactionModeToggle from "./TransactionModeToggle.vue";

const { formatOnSave, selectedTask } = useIssueContext();
const { project } = useCurrentProjectV1();

const database = computed(() => {
  return databaseForTask(project.value, selectedTask.value);
});

const language = useInstanceV1EditorLanguage(
  computed(() => database.value.instanceResource)
);

const shouldShowInstanceRoleSelect = computed(() => {
  // Only works for postgres.
  const engine = database.value.instanceResource.engine;
  if (engine !== Engine.POSTGRES) {
    return false;
  }
  // Only works for DDL/DML, exclude creating database and schema baseline.
  if (
    ![
      Task_Type.DATABASE_SCHEMA_UPDATE,
      Task_Type.DATABASE_DATA_UPDATE,
    ].includes(selectedTask.value.type)
  ) {
    return false;
  }
  return true;
});

const shouldShowTransactionModeToggle = computed(() => {
  // Check if the engine supports transaction mode
  if (
    !instanceV1SupportsTransactionMode(database.value.instanceResource.engine)
  ) {
    return false;
  }
  // Only show for DDL/DML tasks
  if (
    ![
      Task_Type.DATABASE_SCHEMA_UPDATE,
      Task_Type.DATABASE_DATA_UPDATE,
    ].includes(selectedTask.value.type)
  ) {
    return false;
  }
  return true;
});
</script>
