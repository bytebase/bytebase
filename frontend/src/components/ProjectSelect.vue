<template>
  <BBSelect
    :selected-item="state.selectedProject"
    :item-list="projectList"
    :disabled="disabled"
    :placeholder="$t('project.select')"
    :show-prefix-item="true"
    :error="!validate()"
    @select-item="(project: Project) => $emit('select-project-id', project.uid)"
  >
    <template #menuItem="{ item: project }">
      {{ projectV1Name(project) }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive, watch } from "vue";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { State } from "@/types/proto/v1/common";
import { Project, TenantMode } from "@/types/proto/v1/project_service";
import {
  hasWorkspacePermissionV1,
  isMemberOfProjectV1,
  projectV1Name,
} from "@/utils";
import { UNKNOWN_ID, DEFAULT_PROJECT_ID, unknownProject } from "../types";

interface LocalState {
  selectedProject?: Project;
}

export enum Mode {
  Standard = 1,
  Tenant = 2,
}

export default defineComponent({
  name: "ProjectSelect",
  props: {
    selectedId: {
      default: String(UNKNOWN_ID),
      type: String,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    allowedRoleList: {
      default: () => ["OWNER", "DEVELOPER"],
      type: Array as PropType<string[]>,
    },
    includeDefaultProject: {
      default: false,
      type: Boolean,
    },
    mode: {
      type: Number as PropType<Mode>,
      default: Mode.Standard | Mode.Tenant,
    },
    onlyUserself: {
      type: Boolean,
      default: true,
    },
    required: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["select-project-id"],
  setup(props) {
    const state = reactive<LocalState>({
      selectedProject: undefined,
    });

    const currentUserV1 = useCurrentUserV1();
    const projectV1Store = useProjectV1Store();
    const canManageProject = hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-project",
      currentUserV1.value.userRole
    );

    const rawProjectList = computed((): Project[] => {
      let list = projectV1Store.getProjectList(true /* showDeleted */);

      if (props.onlyUserself) {
        list = list.filter((project) => {
          return (
            canManageProject ||
            isMemberOfProjectV1(project.iamPolicy, currentUserV1.value)
          );
        });
      }

      list = list.filter((project) => {
        if (
          project.tenantMode === TenantMode.TENANT_MODE_DISABLED &&
          props.mode & Mode.Standard
        ) {
          return true;
        }
        if (
          project.tenantMode === TenantMode.TENANT_MODE_ENABLED &&
          props.mode & Mode.Tenant
        ) {
          return true;
        }
        return false;
      });

      return list.filter((project: Project) => {
        // Do not show Default project in selector.
        return project.uid !== String(DEFAULT_PROJECT_ID);
      });
    });

    const selectedIdNotInList = computed((): boolean => {
      if (props.selectedId === String(UNKNOWN_ID)) {
        return false;
      }

      return (
        rawProjectList.value.find((item) => {
          return item.uid === props.selectedId;
        }) == null
      );
    });

    const projectList = computed((): Project[] => {
      const list = rawProjectList.value.filter((project) => {
        if (project.state === State.ACTIVE) {
          return true;
        }
        // project.state === State.DELETED
        if (project.uid === props.selectedId) {
          return true;
        }
        return false;
      });

      const defaultProject = projectV1Store.getProjectByUID(
        String(DEFAULT_PROJECT_ID)
      );
      if (
        props.includeDefaultProject ||
        props.selectedId === String(DEFAULT_PROJECT_ID)
      ) {
        // If includeDefaultProject is false but the selected project is the default
        // project, we will show it. If includeDefaultProject is true, then it's
        // already in the list, so no need to show it twice
        list.unshift(defaultProject);
      }

      if (
        props.selectedId !== String(DEFAULT_PROJECT_ID) &&
        selectedIdNotInList.value
      ) {
        // It may happen the selected id might not be in the project list.
        // e.g. the selected project is deleted after the selection and we
        // are unable to cleanup properly. In such case, the selected project id
        // is orphaned and we just display the id
        const dummyProject = {
          ...unknownProject(),
          name: `projects/${props.selectedId}`,
          uid: props.selectedId,
          title: props.selectedId,
        };
        list.unshift(dummyProject);
      }

      return list;
    });

    const validate = () => {
      if (!props.required) {
        return true;
      }
      return (
        !!state.selectedProject &&
        state.selectedProject.uid !== String(UNKNOWN_ID)
      );
    };

    watch(
      [() => props.selectedId, projectList],
      ([selectedId, list]) => {
        state.selectedProject = list.find(
          (project) => project.uid === selectedId
        );
      },
      { immediate: true }
    );

    return {
      UNKNOWN_ID,
      DEFAULT_PROJECT_ID,
      projectV1Name,
      state,
      projectList,
      validate,
      selectedIdNotInList,
    };
  },
});
</script>
