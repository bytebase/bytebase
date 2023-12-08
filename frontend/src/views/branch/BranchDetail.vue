<template>
  <template v-if="ready">
    <template v-if="isCreating">
      <BranchCreateView :project-id="project?.uid" v-bind="$attrs" />
    </template>
    <template v-else-if="branch">
      <BranchDetailView
        v-if="project"
        :key="detailViewKey"
        :project="project"
        :clean-branch="branch.clean"
        :dirty-branch="branch.dirty"
        v-bind="$attrs"
        @update:branch="handleUpdateBranch"
        @update:branch-id="handleUpdateBranchId"
      />
    </template>
  </template>
  <div v-else class="w-full h-full relative">
    <MaskSpinner />
  </div>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { cloneDeep, uniqueId } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import BranchCreateView from "@/components/Branch/BranchCreateView.vue";
import BranchDetailView from "@/components/Branch/BranchDetailView.vue";
import { useProjectV1Store } from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import { Branch } from "@/types/proto/v1/branch_service";
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
const detailViewKey = ref(uniqueId());

const isCreating = computed(() => props.branchName === "new");
const branch = ref<{ clean: Branch; dirty: Branch }>();
const project = computed(() => {
  if (props.projectSlug === "-") {
    return;
  }
  return projectStore.getProjectByUID(
    String(idFromSlug(props.projectSlug as string))
  );
});

const handleUpdateBranch = (br: Branch) => {
  branch.value = {
    clean: br,
    dirty: cloneDeep(br),
  };
  detailViewKey.value = uniqueId();
};
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

    const br = await branchStore.fetchBranchByName(
      branchFullName.value,
      false /* useCache */
    );
    branch.value = {
      clean: br,
      dirty: cloneDeep(br),
    };
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
  if (isCreating.value) {
    return t("schema-designer.new-branch");
  } else {
    if (branch.value) {
      return branch.value.clean.branchId;
    }
  }
  return t("common.loading");
});
useTitle(documentTitle);
</script>
