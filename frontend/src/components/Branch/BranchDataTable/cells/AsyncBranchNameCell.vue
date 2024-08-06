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
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_BRANCH_DETAIL } from "@/router/dashboard/projectV1";
import { useBranchStore } from "@/store";
import {
  getProjectAndBranchId,
} from "@/store/modules/v1/common";

const props = defineProps<{
  name: string;
}>();

const router = useRouter();
const branchStore = useBranchStore();

const branch = computedAsync(async () => {
  const anyView = branchStore.getBranchByName(props.name);
  if (anyView) {
    return anyView;
  }
  return branchStore.fetchBranchByName(props.name);
}, undefined);

const branchName = computed(() => {
  return branch.value?.branchId ?? getProjectAndBranchId(props.name)[1];
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
