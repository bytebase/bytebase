<template>
  <div
    class="w-full grid grid-cols-4 items-center text-sm gap-x-2"
    style="grid-template-columns: 1fr auto 1fr auto"
  >
    <BranchSelector
      class="!w-full text-center"
      :clearable="false"
      :project="project"
      :branch="targetBranch?.name"
      :filter="targetBranchFilter"
      @update:branch="$emit('update:target-branch-name', $event)"
    />
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
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  project: ComposedProject;
  targetBranch: Branch | undefined;
  headBranch: Branch | undefined;
}>();

defineEmits<{
  (event: "update:head-branch-name", branch: string | undefined): void;
  (event: "update:target-branch-name", branch: string | undefined): void;
}>();

const targetBranchFilter = (branch: Branch) => {
  const { headBranch } = props;
  if (!headBranch) {
    return true;
  }
  return branch.engine === headBranch.engine && branch.name !== headBranch.name;
};
const headBranchFilter = (branch: Branch) => {
  const { targetBranch } = props;
  if (!targetBranch) {
    return true;
  }
  return (
    branch.engine === targetBranch.engine && branch.name !== targetBranch.name
  );
};
</script>
