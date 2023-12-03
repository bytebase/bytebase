<template>
  <div class="flex flex-row items-center justify-between gap-x-2 group">
    <div class="flex flex-row items-center gap-x-2">
      <div
        class="flex flex-row items-center gap-x-1 border-b border-transparent group-hover:border-control-border cursor-pointer"
        @click="$emit('click-item', change)"
      >
        <RichDatabaseName
          :database="database"
          :show-instance="false"
          :show-arrow="false"
          :show-production-environment-icon="false"
          tooltip="instance"
        />
        <span>@</span>
        <div>
          {{ branch.branchId }}
        </div>
      </div>
    </div>
    <div
      class="flex flex-row items-center justify-end gap-x-1 invisible group-hover:visible"
    >
      <NButton
        size="small"
        quaternary
        style="--n-padding: 0 6px"
        @click.stop="$emit('remove-item', change)"
      >
        <template #icon>
          <heroicons:x-mark />
        </template>
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { Branch } from "@/types/proto/v1/branch_service";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";

const props = defineProps<{
  change: Change;
  branch: Branch | undefined;
}>();

defineEmits<{
  (event: "click-item", change: Change): void;
  (event: "remove-item", change: Change): void;
}>();

const branch = computed(() => {
  return (
    props.branch ??
    Branch.fromPartial({
      name: props.change.source,
      branchId: "<<Unknown Branch>>",
    })
  );
});

const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(branch.value.baselineDatabase);
});
</script>
