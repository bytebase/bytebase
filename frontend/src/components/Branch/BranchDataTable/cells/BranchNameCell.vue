<template>
  <div class="flex flex-row justify-start items-center gap-1">
    <span
      class="cursor-pointer hover:underline hover:text-blue-600"
      @click.stop="handleBranchClick"
      >{{ branchName }}</span
    >
    <span v-if="isProtected">
      <NTooltip trigger="hover">
        <template #trigger>
          <ShieldAlertIcon class="w-4 h-auto text-gray-500" />
        </template>
        {{ $t("branch.branch-is-protected") }}
      </NTooltip>
    </span>
  </div>
</template>

<script lang="ts" setup>
import { ShieldAlertIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useProjectV1Store } from "@/store";
import {
  getProjectAndBranchId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { useProjectBranchProtectionRules } from "@/store/modules/v1/projectProtectionRoles";
import { Branch } from "@/types/proto/v1/branch_service";
import { projectV1Slug } from "@/utils";
import { wildcardToRegex } from "../../utils";

const props = defineProps<{
  branch: Branch;
}>();

const router = useRouter();
const projectStore = useProjectV1Store();

const project = computed(() => {
  const [projectId] = getProjectAndBranchId(props.branch.name);
  return projectStore.getProjectByName(`${projectNamePrefix}${projectId}`);
});

const branchProtectionRules = useProjectBranchProtectionRules(
  project.value.name
);

const branchName = computed(() => {
  return props.branch.branchId;
});

const isProtected = computed(() => {
  return branchProtectionRules.value.some((rule) => {
    return wildcardToRegex(rule.nameFilter).test(props.branch.branchId);
  });
});

const handleBranchClick = async () => {
  const [_, branchId] = getProjectAndBranchId(props.branch.name);
  router.push({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: projectV1Slug(project.value),
      branchName: `${branchId}`,
    },
  });
};
</script>
