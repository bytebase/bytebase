<template>
  <template v-if="allowArchiveOrRestore">
    <template v-if="project.state === State.ACTIVE">
      <BBButtonConfirm
        :style="'ARCHIVE'"
        :button-text="$t('project.settings.archive.btn-text')"
        :ok-text="$t('common.archive')"
        :confirm-title="
          $t('project.settings.archive.title') + ` '${project.title}'?`
        "
        :confirm-description="$t('project.settings.archive.description')"
        :require-confirm="true"
        @confirm="archiveOrRestoreProject(true)"
      >
        <div class="mt-3">
          <NCheckbox v-model:checked="force">
            <div class="text-sm font-normal text-control-light">
              {{ $t("project.settings.archive.force.description") }}
            </div>
          </NCheckbox>
        </div>
      </BBButtonConfirm>
    </template>
    <template v-else-if="project.state === State.DELETED">
      <BBButtonConfirm
        :style="'RESTORE'"
        :button-text="$t('project.settings.restore.btn-text')"
        :ok-text="$t('common.restore')"
        :confirm-title="
          $t('project.settings.restore.title') + ` '${project.title}'?`
        "
        :confirm-description="''"
        :require-confirm="true"
        @confirm="archiveOrRestoreProject(false)"
      />
    </template>
  </template>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed, ref } from "vue";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { State } from "@/types/proto/v1/common";
import { hasPermissionInProjectV1, hasWorkspacePermissionV1 } from "@/utils";

const props = defineProps<{
  project: ComposedProject;
}>();

const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();

const force = ref(false);

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

const archiveOrRestoreProject = async (archive: boolean) => {
  if (archive) {
    await projectV1Store.archiveProject(props.project, force.value);
  } else {
    await projectV1Store.restoreProject(props.project);
  }
};
</script>
