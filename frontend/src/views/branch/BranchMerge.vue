<template>
  <div class="w-full h-full relative">
    <BranchMergeView
      :project="project"
      :head-branch-name="branchFullName"
      @update:head-branch-name="handleUpdateHeadBranchName"
      @merged="handleMerged"
    />
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
import { useProjectByName } from "@/store";
import { getProjectAndBranchId } from "@/store/modules/v1/common";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  projectId: string;
  branchName: string;
}>();

const router = useRouter();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

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

const handleMerged = (mergedBranch: Branch) => {
  router.replace({
    name: PROJECT_V1_ROUTE_BRANCH_DETAIL,
    params: {
      branchName: mergedBranch.branchId,
    },
  });
};
</script>
