<template>
  <template v-if="ready">
    <template v-if="isCreating">
      <BranchCreateView :project-id="project?.uid" v-bind="$attrs" />
    </template>
    <template v-else-if="branch">
      <BranchDetailView :branch="branch" v-bind="$attrs" />
    </template>
  </template>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import BranchCreateView from "@/components/Branch/BranchCreateView.vue";
import BranchDetailView from "@/components/Branch/BranchDetailView.vue";
import { useProjectV1Store } from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import { idFromSlug } from "@/utils";

const { t } = useI18n();
const route = useRoute();
const projectStore = useProjectV1Store();
const branchStore = useBranchStore();
const branchFullName = ref<string>("");
const ready = ref<boolean>(false);

const isCreating = computed(() => route.params.branchName === "new");
const branch = computed(() => {
  return branchStore.getBranchByName(branchFullName.value);
});
const project = computed(() => {
  if (route.params.projectSlug === "-") {
    return;
  }
  return projectStore.getProjectByUID(
    String(idFromSlug(route.params.projectSlug as string))
  );
});

watch(
  () => route.params,
  async () => {
    if (isCreating.value) {
      return;
    }

    // Prepare branch name from route params.
    const sheetId = (route.params.branchName as string) || "";
    if (!sheetId || !project.value) {
      return;
    }
    branchFullName.value = `${project.value.name}/branches/${sheetId}`;
  },
  {
    immediate: true,
    deep: true,
  }
);

watch(
  () => branchFullName.value,
  async () => {
    ready.value = false;
    if (isCreating.value || !branchFullName.value) {
      ready.value = true;
      return;
    }

    await branchStore.fetchBranchByName(
      branchFullName.value,
      false /* useCache */
    );
    ready.value = true;
  },
  {
    immediate: true,
  }
);

const documentTitle = computed(() => {
  if (isCreating.value) {
    return t("schema-designer.new-branch");
  } else {
    if (branch.value) {
      return branch.value.branchId;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
