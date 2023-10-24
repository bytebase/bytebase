<template>
  <ArchiveBanner v-if="project.state === State.DELETED" class="py-2" />
  <HideInStandaloneMode>
    <BBAttention
      v-if="isDefaultProject"
      class="mx-6 mb-4"
      :style="'INFO'"
      :title="$t('project.overview.info-slot-content')"
    />
  </HideInStandaloneMode>

  <div class="py-4 px-6">
    <router-view
      :project-slug="projectSlug"
      :project-webhook-slug="projectWebhookSlug"
      :allow-edit="allowEdit"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import HideInStandaloneMode from "@/components/misc/HideInStandaloneMode.vue";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_V1_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  idFromSlug,
  hasWorkspacePermissionV1,
  hasPermissionInProjectV1,
} from "@/utils";

const props = defineProps({
  projectSlug: {
    required: true,
    type: String,
  },
  projectWebhookSlug: {
    type: String,
    default: undefined,
  },
});

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();

const project = computed(() => {
  return projectV1Store.getProjectByUID(String(idFromSlug(props.projectSlug)));
});

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const isTenantProject = computed((): boolean => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
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
</script>
