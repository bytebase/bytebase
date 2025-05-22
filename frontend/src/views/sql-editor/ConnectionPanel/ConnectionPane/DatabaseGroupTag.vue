<template>
  <NTooltip v-if="databaseGroup">
    <template #trigger>
      <NTag closable @close="() => $emit('uncheck', databaseGroupName)">
        {{ databaseGroup.title }}
      </NTag>
    </template>
    {{ $t("common.database-group") }}
  </NTooltip>
</template>

<script lang="tsx" setup>
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useDatabaseGroupByName } from "@/store";

const props = defineProps<{
  databaseGroupName: string;
}>();

defineEmits<{
  (event: "uncheck", databaseGroupName: string): void;
}>();

const { databaseGroup } = useDatabaseGroupByName(
  computed(() => props.databaseGroupName)
);
</script>
