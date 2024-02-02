<template>
  <div class="relative text-sm">
    <div class="flex flex-row items-center gap-x-1">
      <span v-if="!headBranch || !targetBranch">
        {{ $t("branch.merge-rebase.select-branches-to-merge") }}
      </span>
      <template v-else>
        <template v-if="isValidating">
          <BBSpin class="!w-4 !h-4" />
          <span>{{ $t("branch.merge-rebase.validating-branch") }}</span>
        </template>
        <template v-else-if="validationState">
          <template v-if="validationState.status === Status.OK">
            <CheckIcon class="w-4 h-4 text-success" />
            <span>{{ $t("branch.merge-rebase.able-to-merge") }}</span>
          </template>
          <template v-else>
            <div class="flex flex-row items-center gap-x-1">
              <XCircleIcon class="w-4 h-4 text-error" />
              <i18n-t
                keypath="branch.merge-rebase.cannot-automatically-merge"
                tag="span"
                class="text-error"
              >
                <template #rebase_branch>
                  <router-link
                    :to="rebaseLink(headBranch, targetBranch)"
                    class="normal-link"
                  >
                    {{ $t("branch.merge-rebase.go-rebase") }}
                  </router-link>
                </template>
              </i18n-t>
            </div>
          </template>
        </template>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckIcon, XCircleIcon } from "lucide-vue-next";
import { Status } from "nice-grpc-common";
import { PROJECT_V1_ROUTE_BRANCH_REBASE } from "@/router/dashboard/projectV1";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { MergeBranchValidationState } from "./types";

defineProps<{
  project: ComposedProject;
  targetBranch: Branch | undefined;
  headBranch: Branch | undefined;
  isLoadingTargetBranch?: boolean;
  isLoadingHeadBranch?: boolean;
  isValidating?: boolean;
  validationState: MergeBranchValidationState | undefined;
}>();

const rebaseLink = (headBranch: Branch, targetBranch: Branch) => {
  return {
    name: PROJECT_V1_ROUTE_BRANCH_REBASE,
    params: {
      branchName: headBranch.branchId,
    },
    query: {
      source: targetBranch.branchId,
    },
  };
};
</script>
