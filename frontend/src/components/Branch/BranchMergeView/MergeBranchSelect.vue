<template>
  <div
    class="w-full grid grid-cols-3 items-end text-sm gap-x-2"
    style="grid-template-columns: 1fr auto 1fr auto"
  >
    <div class="flex flex-col overflow-x-hidden">
      <div v-if="parentBranchOnly">
        {{ $t("schema-designer.parent-branch") }}
      </div>
      <NElement
        tag="div"
        class="flex overflow-x-hidden items-center gap-x-1"
        style="
          padding: 0 6px 0 12px;
          border: 1px solid var(--border-color);
          border-radius: var(--border-radius);
          min-height: var(--height-medium);
        "
      >
        <InstanceV1EngineIcon
          v-if="parentDatabase"
          :instance="parentDatabase.instanceResource"
        />
        <NPerformantEllipsis>
          {{ targetBranch?.branchId }}
        </NPerformantEllipsis>
      </NElement>
    </div>
    <div class="flex flex-row justify-center px-2">
      <MoveLeftIcon :size="40" stroke-width="1" />
    </div>
    <BranchSelector
      class="!full text-center"
      :clearable="false"
      :project="project"
      :branch="headBranch?.name"
      :filter="headBranchFilter"
      @update:branch="$emit('update:head-branch-name', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { MoveLeftIcon } from "lucide-vue-next";
import { NElement, NPerformantEllipsis } from "naive-ui";
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName, type ComposedProject } from "@/types";
import type { Branch } from "@/types/proto/v1/branch_service";
import BranchSelector from "../BranchSelector.vue";

const props = defineProps<{
  project: ComposedProject;
  targetBranch: Branch | undefined;
  headBranch: Branch | undefined;
  parentBranchOnly: boolean | undefined;
}>();

defineEmits<{
  (event: "update:head-branch-name", branch: string | undefined): void;
  (event: "update:target-branch-name", branch: string | undefined): void;
}>();

const parentDatabase = computed(() => {
  const name = props.targetBranch?.baselineDatabase;
  if (!isValidDatabaseName(name)) return undefined;
  return useDatabaseV1Store().getDatabaseByName(name);
});

const headBranchFilter = (branch: Branch) => {
  const { targetBranch } = props;
  if (!targetBranch) {
    return true;
  }
  return (
    branch.engine === targetBranch.engine &&
    branch.name !== targetBranch.name &&
    // Main branches are not allow be merged to any branch.
    !!branch.parentBranch
  );
};
</script>
