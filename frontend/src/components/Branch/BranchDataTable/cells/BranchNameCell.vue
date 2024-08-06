<template>
  <div class="flex flex-row justify-start items-center gap-1">
    <span
      class="cursor-pointer hover:underline hover:text-blue-600"
      @click.stop="handleBranchClick"
      >{{ branchName }}</span
    >
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_BRANCH_DETAIL } from "@/router/dashboard/projectV1";
import {
  getProjectAndBranchId,
} from "@/store/modules/v1/common";
import type { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  branch: Branch;
}>();

const router = useRouter();

const branchName = computed(() => {
  return props.branch.branchId;
});

const handleBranchClick = async () => {
  const [_, branchId] = getProjectAndBranchId(props.branch.name);
  router.push({
    name: PROJECT_V1_ROUTE_BRANCH_DETAIL,
    params: {
      branchName: `${branchId}`,
    },
  });
};
</script>
