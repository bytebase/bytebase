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
      <FormatOnSaveCheckbox v-model:value="formatOnSave" :language="language" />
    </div>
  </NPopover>
</template>

<script setup lang="tsx">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { NPopover, NButton, NDivider } from "naive-ui";
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { Engine } from "@/types/proto/v1/common";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { useInstanceV1EditorLanguage } from "@/utils";
import FormatOnSaveCheckbox from "./FormatOnSaveCheckbox.vue";
import InstanceRoleSelect from "./InstanceRoleSelect.vue";

const { formatOnSave, issue, selectedTask } = useIssueContext();

const database = computed(() => {
  return databaseForTask(issue.value.projectEntity, selectedTask.value);
});

const language = useInstanceV1EditorLanguage(
  computed(() => database.value.instanceResource)
);

const shouldShowInstanceRoleSelect = computed(() => {
  // Only works for postgres.
  if (![Engine.POSTGRES].includes(database.value.instanceResource.engine)) {
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
</script>
