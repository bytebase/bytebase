<template>
  <div class="max-w-3xl mx-auto space-y-4">
    <div class="divide-y divide-block-border space-y-6">
      <ProjectGeneralSettingPanel :project="project" :allow-edit="allowEdit" />
      <div class="pt-4">
        <ProjectMemberPanel :project="project" />
      </div>
    </div>
    <template v-if="allowArchiveOrRestore">
      <template v-if="project.rowStatus == 'NORMAL'">
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
      <template v-else-if="project.rowStatus == 'ARCHIVED'">
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

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { useStore } from "vuex";
import { isProjectOwner } from "../utils";
import ProjectGeneralSettingPanel from "../components/ProjectGeneralSettingPanel.vue";
import ProjectMemberPanel from "../components/ProjectMemberPanel.vue";
import { ProjectPatch, Project } from "../types";
import { useCurrentUser, useProjectStore } from "@/store";

export default defineComponent({
  name: "ProjectSettingPanel",
  components: {
    ProjectGeneralSettingPanel,
    ProjectMemberPanel,
  },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    allowEdit: {
      default: true,
      type: Boolean,
    },
  },
  setup(props) {
    const store = useStore();

    const currentUser = useCurrentUser();
    const projectStore = useProjectStore();

    // Only the project owner can archive/restore the project info.
    // This means even the workspace owner won't be able to edit it.
    // There seems to be no good reason that workspace owner needs to archive/restore the project.
    const allowArchiveOrRestore = computed(() => {
      for (const member of props.project.memberList) {
        if (member.principal.id == currentUser.value.id) {
          if (isProjectOwner(member.role)) {
            return true;
          }
        }
      }
      return false;
    });

    const archiveOrRestoreProject = (archive: boolean) => {
      const projectPatch: ProjectPatch = {
        rowStatus: archive ? "ARCHIVED" : "NORMAL",
      };
      projectStore.patchProject({
        projectId: props.project.id,
        projectPatch,
      });
    };

    return {
      allowArchiveOrRestore,
      archiveOrRestoreProject,
    };
  },
});
</script>
