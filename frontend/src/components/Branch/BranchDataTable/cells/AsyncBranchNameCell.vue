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
import { computedAsync } from "@vueuse/core";
import { ShieldAlertIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_BRANCH_DETAIL } from "@/router/dashboard/projectV1";
import { useBranchStore, useProjectV1Store } from "@/store";
import {
  getProjectAndBranchId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { useProjectBranchProtectionRules } from "@/store/modules/v1/projectProtectionRoles";
import { wildcardToRegex } from "../../utils";

const props = defineProps<{
  name: string;
}>();

const router = useRouter();
const projectStore = useProjectV1Store();
const branchStore = useBranchStore();

const branch = computedAsync(async () => {
  const anyView = branchStore.getBranchByName(props.name);
  if (anyView) {
    return anyView;
  }
  return branchStore.fetchBranchByName(props.name);
}, undefined);

const project = computed(() => {
  const [projectId] = getProjectAndBranchId(props.name);
  return projectStore.getProjectByName(`${projectNamePrefix}${projectId}`);
});

const branchProtectionRules = useProjectBranchProtectionRules(
  project.value.name
);

const branchName = computed(() => {
  return branch.value?.branchId ?? getProjectAndBranchId(props.name)[1];
});

const isProtected = computed(() => {
  return branchProtectionRules.value.some((rule) => {
    return wildcardToRegex(rule.nameFilter).test(props.name);
  });
});

const handleBranchClick = async () => {
  const [_, branchId] = getProjectAndBranchId(props.name);
  router.push({
    name: PROJECT_V1_ROUTE_BRANCH_DETAIL,
    params: {
      branchName: `${branchId}`,
    },
  });
};
</script>
