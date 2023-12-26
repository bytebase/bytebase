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
import { useProjectV1Store } from "@/store";
import { getProjectAndBranchId } from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";
import { idFromSlug } from "@/utils";

const props = defineProps<{
  projectSlug: string;
  branchName: string;
}>();

const router = useRouter();

const project = computed(() => {
  if (props.projectSlug === "-") {
    return;
  }
  return useProjectV1Store().getProjectByUID(
    String(idFromSlug(props.projectSlug as string))
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
    name: "workspace.project.branch.rebase",
    params: {
      projectSlug: props.projectSlug,
      branchName: branchId,
    },
    query: router.currentRoute.value.query,
    hash: router.currentRoute.value.hash,
  });
};

const handleRebased = (
  rebasedBranch: Branch,
  headBranchName: string,
  headBranch: Branch | undefined
) => {
  router.replace({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: props.projectSlug,
      branchName: rebasedBranch.branchId,
    },
  });
};
</script>
