<template>
  <div
    v-if="allowArchiveOrRestore"
    class="w-full flex items-center justify-between"
  >
    <template v-if="project.state === State.ACTIVE">
      <BBButtonConfirm
        :type="'ARCHIVE'"
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
              {{ $t("instance.force-archive-description") }}
            </div>
          </NCheckbox>
        </div>
      </BBButtonConfirm>
    </template>
    <template v-else-if="project.state === State.DELETED">
      <BBButtonConfirm
        :type="'RESTORE'"
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
    <ResourceHardDeleteButton :resource="project" @delete="hardDeleteProject" />
  </div>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { BBButtonConfirm } from "@/bbkit";
import ResourceHardDeleteButton from "@/components/v2/Button/ResourceHardDeleteButton.vue";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { SETTING_ROUTE_WORKSPACE_ARCHIVE } from "@/router/dashboard/workspaceSetting";
import { useProjectV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  project: ComposedProject;
}>();

const projectV1Store = useProjectV1Store();
const router = useRouter();
const force = ref(false);

const allowArchiveOrRestore = computed(() => {
  if (props.project.state === State.ACTIVE) {
    return hasWorkspacePermissionV2("bb.projects.delete");
  }
  return hasWorkspacePermissionV2("bb.projects.undelete");
});

const archiveOrRestoreProject = async (archive: boolean) => {
  if (archive) {
    await projectV1Store.archiveProject(props.project, force.value);
  } else {
    await projectV1Store.restoreProject(props.project);
  }

  if (archive) {
    router.replace({
      name: PROJECT_V1_ROUTE_DASHBOARD,
    });
  }
};

const hardDeleteProject = async (resource: string) => {
  await projectV1Store.deleteProject(resource);
  router.replace({
    name: SETTING_ROUTE_WORKSPACE_ARCHIVE,
    hash: "#PROJECT",
  });
};
</script>
