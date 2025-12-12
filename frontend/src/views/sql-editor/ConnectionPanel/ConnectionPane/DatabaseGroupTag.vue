<template>
  <NTooltip v-if="databaseGroup">
    <template #trigger>
      <NTag closable :disabled="disabled" @close="() => $emit('uncheck', databaseGroupName)">
        <div class="flex items-center gap-x-1">
          <BoxesIcon class="w-4" />
          {{ databaseGroup.title }}
        </div>
      </NTag>
    </template>
    {{ $t("common.database-group") }}
  </NTooltip>
</template>

<script lang="tsx" setup>
import { BoxesIcon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useDatabaseGroupByName } from "@/store";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";

const props = defineProps<{
  databaseGroupName: string;
  disabled: boolean;
}>();

defineEmits<{
  (event: "uncheck", databaseGroupName: string): void;
}>();

const { databaseGroup } = useDatabaseGroupByName(
  computed(() => props.databaseGroupName),
  DatabaseGroupView.FULL
);
</script>
