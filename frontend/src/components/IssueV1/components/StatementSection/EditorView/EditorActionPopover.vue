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
import { useCurrentProjectV1 } from "@/store";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { useInstanceV1EditorLanguage } from "@/utils";
import { convertEngineToNew } from "@/utils/v1/common-conversions";
import FormatOnSaveCheckbox from "./FormatOnSaveCheckbox.vue";
import InstanceRoleSelect from "./InstanceRoleSelect.vue";

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
  const engine = convertEngineToNew(database.value.instanceResource.engine);
  if (![Engine.POSTGRES].includes(engine)) {
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
