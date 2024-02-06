<template>
  <div
    class="w-full grid grid-cols-3 items-end text-sm gap-x-2"
    style="grid-template-columns: 1fr auto 1fr"
  >
    <BranchSelector
      class="!full text-center"
      :clearable="false"
      :project="project"
      :branch="headBranch?.name"
      :filter="headBranchFilter"
      @update:branch="$emit('update:head-branch-name', $event)"
    />
    <div class="flex flex-row justify-center px-2 h-[34px]">
      <MoveLeftIcon :size="40" stroke-width="1" />
    </div>
    <div class="flex flex-col">
      <NRadioGroup
        :value="sourceType"
        class="space-x-2"
        @update:value="$emit('update:source-type', $event as RebaseSourceType)"
      >
        <NRadio value="BRANCH">{{ $t("common.branch") }}</NRadio>
        <NRadio value="DATABASE">{{ $t("common.database") }}</NRadio>
      </NRadioGroup>
      <BranchSelector
        v-if="sourceType === 'BRANCH'"
        class="!w-full text-center"
        :clearable="false"
        :project="project"
        :branch="sourceBranch?.name"
        :filter="sourceBranchFilter"
        @update:branch="$emit('update:source-branch-name', $event)"
      />
      <DatabaseSelect
        v-if="sourceType === 'DATABASE'"
        :database="sourceDatabase?.uid"
        :project="project.uid"
        :allowed-engine-type-list="headBranch ? [headBranch.engine] : undefined"
        style="width: 100%"
        @update:database="$emit('update:source-database-uid', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { MoveLeftIcon } from "lucide-vue-next";
import { NRadioGroup } from "naive-ui";
import { computed } from "vue";
import { DatabaseSelect } from "@/components/v2";
import { ComposedDatabase, ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { RebaseSourceType } from "./types";

const props = defineProps<{
  project: ComposedProject;
  sourceType: RebaseSourceType;
  headBranch: Branch | undefined;
  sourceBranch: Branch | undefined;
  sourceDatabase: ComposedDatabase | undefined;
  isLoadingSourceBranch?: boolean;
  isLoadingHeadBranch?: boolean;
}>();

defineEmits<{
  (event: "update:source-type", type: RebaseSourceType): void;
  (event: "update:head-branch-name", branch: string | undefined): void;
  (event: "update:source-branch-name", branch: string | undefined): void;
  (event: "update:source-database-uid", uid: string | undefined): void;
}>();

const sourceBranchOrDatabase = computed(() => {
  return props.sourceType === "BRANCH"
    ? props.sourceBranch
    : props.sourceDatabase;
});

const sourceBranchFilter = (branch: Branch) => {
  const { headBranch } = props;
  if (!headBranch) {
    return true;
  }
  return branch.engine === headBranch.engine && branch.name !== headBranch.name;
};
const headBranchFilter = (branch: Branch) => {
  const source = sourceBranchOrDatabase.value;
  if (!source) {
    return true;
  }
  if (props.sourceType === "BRANCH") {
    const sourceBranch = source as Branch;
    return (
      branch.engine === sourceBranch.engine && branch.name !== sourceBranch.name
    );
  } else {
    const sourceDatabase = source as ComposedDatabase;
    return branch.engine === sourceDatabase.instanceEntity.engine;
  }
};
</script>
