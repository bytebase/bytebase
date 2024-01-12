<template>
  <div class="w-full h-full relative">
    <BranchRebaseView
      v-if="project"
      :project="project"
      :head-branch-name="branchFullName"
      @update:head-branch-name="handleUpdateHeadBranchName"
      @rebased="handleRebased"
    />
    <MaskSpinner v-else />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import BranchRebaseView from "@/components/Branch/BranchRebaseView";
import {
  PROJECT_V1_BRANCHE_REBASE,
  PROJECT_V1_BRANCHE_DETAIL,
} from "@/router/dashboard/projectV1";
import { useProjectV1Store } from "@/store";
import { getProjectAndBranchId } from "@/store/modules/v1/common";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  projectId: string;
  branchName: string;
}>();

const router = useRouter();

const project = computed(() => {
  if (props.projectId === "-") {
    return;
  }
  return useProjectV1Store().getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const branchFullName = computed(() => {
  if (!project.value) return null;
  if (props.branchName === "-") return null;
  return `${project.value.name}/branches/${props.branchName}`;
});

const handleUpdateHeadBranchName = (branchName: string | null) => {
  const branchId = branchName ? getProjectAndBranchId(branchName)[1] : "-";
  router.replace({
    name: PROJECT_V1_BRANCHE_REBASE,
    params: {
      branchName: branchId,
    },
    query: router.currentRoute.value.query,
    hash: router.currentRoute.value.hash,
  });
};

const handleRebased = (rebasedBranch: Branch) => {
  router.replace({
    name: PROJECT_V1_BRANCHE_DETAIL,
    params: {
      branchName: rebasedBranch.branchId,
    },
  });
};
</script>
