<template>
  <ArchiveBanner v-if="project.state === State.DELETED" class="py-2" />
  <div class="p-6 h-full overflow-auto">
    <HideInStandaloneMode>
      <template v-if="isDefaultProject">
        <h1 class="mb-4 text-xl font-bold leading-6 text-main truncate">
          {{ $t("database.unassigned-databases") }}
        </h1>
      </template>
      <BBAttention
        v-if="isDefaultProject"
        class="mb-4"
        :style="'INFO'"
        :title="$t('project.overview.info-slot-content')"
      />
    </HideInStandaloneMode>
    <router-view :project-id="projectId" :allow-edit="allowEdit" />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted } from "vue";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import HideInStandaloneMode from "@/components/misc/HideInStandaloneMode.vue";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV1, hasPermissionInProjectV1 } from "@/utils";

const props = defineProps({
  projectId: {
    required: true,
    type: String,
  },
});

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const allowEdit = computed(() => {
  if (project.value.state === State.DELETED) {
    return false;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-project",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.manage-general"
    )
  ) {
    return true;
  }
  return false;
});

onMounted(async () => {
  await projectV1Store.getOrFetchProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});
</script>
