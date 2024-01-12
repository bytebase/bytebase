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
        :readonly="!allowEdit"
        v-bind="$attrs"
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
import { PROJECT_V1_ROUTE_BRANCH_DETAIL } from "@/router/dashboard/projectV1";
import { extractUserEmail, useCurrentUserV1, useProjectV1Store } from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";
import { isOwnerOfProjectV1 } from "@/utils";

const props = defineProps<{
  projectId: string;
  branchName: string;
}>();

const { t } = useI18n();
const router = useRouter();
const me = useCurrentUserV1();
const projectStore = useProjectV1Store();
const branchStore = useBranchStore();
const branchFullName = ref<string>("");
const ready = ref<boolean>(false);
const detailViewKey = ref(uniqueId());

const isCreating = computed(() => props.branchName === "new");
const branch = ref<{ clean: Branch; dirty: Branch }>();
const project = computed(() => {
  if (props.projectId === "-") {
    return;
  }
  return projectStore.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});
const allowEdit = computed(() => {
  if (!project.value) return false;
  if (!branch.value) return false;
  if (isOwnerOfProjectV1(project.value.iamPolicy, me.value)) {
    return true;
  }
  return extractUserEmail(branch.value.clean.creator) === me.value.email;
});

const handleUpdateBranchId = (id: string) => {
  if (!branch.value) return;
  branch.value.clean.branchId = id;
  branch.value.dirty.branchId = id;
  router.replace({
    name: PROJECT_V1_ROUTE_BRANCH_DETAIL,
    params: {
      branchName: id,
    },
  });
};

watch(
  [() => props.projectId, () => props.branchName],
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
