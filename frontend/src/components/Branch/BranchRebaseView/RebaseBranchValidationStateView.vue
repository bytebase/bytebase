<template>
  <div class="flex flex-row items-center gap-x-1">
    <span v-if="!headBranch || !sourceBranchOrDatabase">
      {{ $t("branch.merge-rebase.select-branches-to-rebase") }}
    </span>
    <template v-else>
      <template v-if="isValidating">
        <BBSpin class="!w-4 !h-4" />
        <span>{{ $t("branch.merge-rebase.validating-branch") }}</span>
      </template>
      <template v-else-if="validationState">
        <template v-if="validationState.branch">
          <CheckIcon class="w-4 h-4 text-success" />
          <span>{{ $t("branch.merge-rebase.able-to-rebase") }}</span>
        </template>
        <template v-else>
          <XCircleIcon class="w-4 h-4 text-error" />
          <span class="text-error">
            {{ $t("branch.merge-rebase.cannot-automatically-rebase") }}
          </span>
        </template>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import { CheckIcon, XCircleIcon } from "lucide-vue-next";
import { computed } from "vue";
import { ComposedDatabase, ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { RebaseBranchValidationState, RebaseSourceType } from "./types";

const props = defineProps<{
  project: ComposedProject;
  sourceType: RebaseSourceType;
  headBranch: Branch | undefined;
  sourceBranch: Branch | undefined;
  sourceDatabase: ComposedDatabase | undefined;
  isLoadingSourceBranch?: boolean;
  isLoadingHeadBranch?: boolean;
  isValidating?: boolean;
  validationState: RebaseBranchValidationState | undefined;
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
</script>
