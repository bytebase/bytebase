<template>
  <div class="w-full h-full relative">
    <BranchMergeView
      v-if="project"
      :project="project"
      :head-branch-name="branchFullName"
      @update:head-branch-name="handleUpdateHeadBranchName"
      @merged="handleMerged"
    />
    <MaskSpinner v-else />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import BranchMergeView from "@/components/Branch/BranchMergeView";
import {
  PROJECT_V1_ROUTE_BRANCH_MERGE,
  PROJECT_V1_ROUTE_BRANCH_DETAIL,
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
    name: PROJECT_V1_ROUTE_BRANCH_MERGE,
    params: {
      branchName: branchId,
    },
    query: router.currentRoute.value.query,
    hash: router.currentRoute.value.hash,
  });
};

const handleMerged = (
  mergedBranch: Branch,
  headBranchName: string,
  headBranch: Branch | undefined
) => {
  router.replace({
    name: PROJECT_V1_ROUTE_BRANCH_DETAIL,
    params: {
      branchName: mergedBranch.branchId,
    },
  });
};
</script>
