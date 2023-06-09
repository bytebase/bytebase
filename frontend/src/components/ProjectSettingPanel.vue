<template>
  <div class="max-w-3xl mx-auto space-y-4">
    <div class="divide-y divide-block-border space-y-6">
      <ProjectGeneralSettingPanel :project="project" :allow-edit="allowEdit" />
    </div>
    <template v-if="allowArchiveOrRestore">
      <template v-if="project.state === State.ACTIVE">
        <BBButtonConfirm
          :style="'ARCHIVE'"
          :button-text="$t('project.settings.archive.btn-text')"
          :ok-text="$t('common.archive')"
          :confirm-title="
            $t('project.settings.archive.title') + ` '${project.name}'?`
          "
          :confirm-description="$t('project.settings.archive.description')"
          :require-confirm="true"
          @confirm="archiveOrRestoreProject(true)"
        />
      </template>
      <template v-else-if="project.state === State.DELETED">
        <BBButtonConfirm
          :style="'RESTORE'"
          :button-text="$t('project.settings.restore.btn-text')"
          :ok-text="$t('common.restore')"
          :confirm-title="
            $t('project.settings.restore.title') + ` '${project.name}'?`
          "
          :confirm-description="''"
          :require-confirm="true"
          @confirm="archiveOrRestoreProject(false)"
        />
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { hasPermissionInProjectV1, hasWorkspacePermissionV1 } from "../utils";
import ProjectGeneralSettingPanel from "../components/Project/ProjectGeneralSettingPanel.vue";
import {
  useCurrentUserV1,
  useGracefulRequest,
  useProjectV1Store,
} from "@/store";
import { ComposedProject } from "@/types";
import { State } from "@/types/proto/v1/common";

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<ComposedProject>,
  },
  allowEdit: {
    default: true,
    type: Boolean,
  },
});

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();

const allowArchiveOrRestore = computed(() => {
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
      props.project.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.manage-general"
    )
  ) {
    return true;
  }
  return false;
});

const archiveOrRestoreProject = (archive: boolean) => {
  useGracefulRequest(async () => {
    if (archive) {
      await projectV1Store.archiveProject(props.project);
    } else {
      await projectV1Store.restoreProject(props.project);
    }
  });
};
</script>
