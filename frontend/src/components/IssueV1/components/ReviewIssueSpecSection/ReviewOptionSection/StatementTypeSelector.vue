<template>
  <NRadioGroup
    v-if="editable"
    :value="statementType"
    @update:value="handleUpdate"
  >
    <NRadio
      v-for="item in availableStatementTypes"
      :key="item"
      :value="item"
      :label="stringifyStatementType(item)"
    />
  </NRadioGroup>
  <template v-else>
    <span class="text-base">{{ stringifyStatementType(statementType) }}</span>
  </template>
</template>

<script setup lang="ts">
import { NRadioGroup, NRadio } from "naive-ui";
import { computed } from "vue";
import { onMounted } from "vue";
import { Plan_ChangeDatabaseConfig_Type } from "@/types/proto/v1/plan_service";
import type { StatementType } from "./types";

const props = defineProps<{
  statementType: StatementType;
  editable?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:statement-type", value: StatementType): void;
}>();

const availableStatementTypes = computed((): StatementType[] => [
  Plan_ChangeDatabaseConfig_Type.MIGRATE,
  Plan_ChangeDatabaseConfig_Type.DATA,
]);

const handleUpdate = (value: StatementType) => {
  emit("update:statement-type", value);
};

const stringifyStatementType = (statementType: StatementType) => {
  if (statementType === Plan_ChangeDatabaseConfig_Type.MIGRATE) {
    return "DDL";
  } else if (statementType === Plan_ChangeDatabaseConfig_Type.DATA) {
    return "DML";
  } else {
    return "Unknown";
  }
};

onMounted(() => {
  if (!availableStatementTypes.value.includes(props.statementType)) {
    handleUpdate(availableStatementTypes.value[0]);
  }
});
</script>
