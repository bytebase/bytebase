<template>
  <div class="w-full h-full relative">
    <BranchRolloutView
      v-if="project && ready && branch"
      :project="project"
      :branch="branch"
      v-bind="$attrs"
    />
    <MaskSpinner v-else />
  </div>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import BranchRolloutView from "@/components/Branch/BranchRolloutView";
import { useProjectV1Store } from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";

const props = defineProps<{
  projectId: string;
  branchName: string;
}>();

const { t } = useI18n();
const router = useRouter();
const projectStore = useProjectV1Store();
const branchStore = useBranchStore();
const branchFullName = ref<string>("");
const ready = ref<boolean>(false);

const project = computed(() => {
  if (props.projectId === "-") {
    return;
  }
  return projectStore.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});
const branch = ref<Branch>();

watch(
  [() => props.projectId, () => props.branchName],
  async () => {
    // Prepare branch name from route params.
    const branchId = props.branchName;
    if (!branchId || !project.value) {
      return;
    }
    branchFullName.value = `${project.value.name}/branches/${branchId}`;
  },
  {
    immediate: true,
  }
);

watch(
  () => branchFullName.value,
  async () => {
    ready.value = false;
    const br = await branchStore.fetchBranchByName(
      branchFullName.value,
      false /* useCache */
    );
    branch.value = br;
    if (!br) {
      router.replace("error.404");
    }
    ready.value = true;
  },
  {
    immediate: true,
  }
);

const documentTitle = computed(() => {
  if (branch.value) {
    return branch.value.branchId;
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
