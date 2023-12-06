<template>
  <template v-if="ready">
    <template v-if="isCreating">
      <BranchCreateView :project-id="project?.uid" v-bind="$attrs" />
    </template>
    <template v-else-if="branch">
      <BranchDetailView
        :project-id="getProjectName(project?.name ?? '')"
        :branch="branch"
        v-bind="$attrs"
        @update:branch="branch = $event"
        @update:branch-id="handleUpdateBranchId"
      />
    </template>
  </template>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import BranchCreateView from "@/components/Branch/BranchCreateView.vue";
import BranchDetailView from "@/components/Branch/BranchDetailView.vue";
import { useProjectV1Store } from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import { getProjectName } from "@/store/modules/v1/common";
import { idFromSlug } from "@/utils";

const props = defineProps<{
  projectSlug: string;
  branchName: string;
}>();

const { t } = useI18n();
const router = useRouter();
const projectStore = useProjectV1Store();
const branchStore = useBranchStore();
const branchFullName = ref<string>("");
const ready = ref<boolean>(false);

const isCreating = computed(() => props.branchName === "new");
const branch = computed(() => {
  return branchStore.getBranchByName(branchFullName.value);
});
const project = computed(() => {
  if (props.projectSlug === "-") {
    return;
  }
  return projectStore.getProjectByUID(
    String(idFromSlug(props.projectSlug as string))
  );
});

const handleUpdateBranchId = (id: string) => {
  router.replace({
    params: {
      projectSlug: props.projectSlug,
      branchName: id,
    },
  });
};

watch(
  [() => props.projectSlug, () => props.branchName],
  async () => {
    if (isCreating.value) {
      return;
    }

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
    if (isCreating.value || !branchFullName.value) {
      ready.value = true;
      return;
    }

    const branch = await branchStore.fetchBranchByName(
      branchFullName.value,
      false /* useCache */
    );
    if (!branch) {
      router.replace("error.404");
    }
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
