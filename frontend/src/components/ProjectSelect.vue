<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e: any) => {
        $emit('select-project-id', parseInt(e.target.value, 10));
      }
    "
  >
    <option disabled :selected="UNKNOWN_ID == state.selectedId">
      Select project
    </option>
    <!-- If includeDefaultProject is false but the selected project is the default
         project, we will show it. If includeDefaultProject is true, then it's
    already in the list, so no need to show it twice-->
    <option
      v-if="!includeDefaultProject && state.selectedId == DEFAULT_PROJECT_ID"
      :selected="true"
    >
      Default
    </option>
    <!-- It may happen the selected id might not be in the project list.
         e.g. the selected project is deleted after the selection and we
         are unable to cleanup properly. In such case, the seleted project id
    is orphaned and we just display the id.-->
    <option
      v-else-if="selectedIdNotInList"
      :value="state.selectedId"
      :selected="true"
    >
      {{ state.selectedId }}
    </option>
    <template v-for="(project, index) in projectList" :key="index">
      <option
        v-if="project.rowStatus == 'NORMAL'"
        :value="project.id"
        :selected="project.id == state.selectedId"
      >
        {{ projectName(project) }}
      </option>
      <option
        v-else-if="project.id == state.selectedId"
        :value="project.id"
        :selected="true"
      >
        {{ projectName(project) }}
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  PropType,
  reactive,
  watch,
  watchEffect,
} from "vue";
import { useStore } from "vuex";
import {
  Project,
  UNKNOWN_ID,
  DEFAULT_PROJECT_ID,
  ProjectRoleType,
} from "../types";
import { isDBAOrOwner } from "../utils";
import { featureToRef, useCurrentUser } from "@/store";

interface LocalState {
  selectedId: number;
}

export enum Mode {
  Standard = 1,
  Tenant = 2,
}

export default defineComponent({
  name: "ProjectSelect",
  props: {
    selectedId: {
      default: UNKNOWN_ID,
      type: Number,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
    allowedRoleList: {
      default: () => ["OWNER", "DEVELOPER"],
      type: Array as PropType<ProjectRoleType[]>,
    },
    includeDefaultProject: {
      default: false,
      type: Boolean,
    },
    mode: {
      type: Number as PropType<Mode>,
      default: Mode.Standard | Mode.Tenant,
    },
  },
  emits: ["select-project-id"],
  setup(props) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const currentUser = useCurrentUser();

    const prepareProjectList = () => {
      store.dispatch("project/fetchProjectListByUser", {
        userId: currentUser.value.id,
        rowStatusList: ["NORMAL", "ARCHIVED"],
      });
    };

    watchEffect(prepareProjectList);

    const hasRBACFeature = featureToRef("bb.feature.rbac");

    const projectList = computed((): Project[] => {
      let list = store.getters["project/projectListByUser"](
        currentUser.value.id,
        ["NORMAL", "ARCHIVED"]
      ) as Project[];

      if (props.includeDefaultProject) {
        list.unshift(store.getters["project/projectById"](DEFAULT_PROJECT_ID));
      }

      list = list.filter((project) => {
        if (project.tenantMode === "DISABLED" && props.mode & Mode.Standard) {
          return true;
        }
        if (project.tenantMode === "TENANT" && props.mode & Mode.Tenant) {
          return true;
        }
        return false;
      });

      if (!hasRBACFeature.value || isDBAOrOwner(currentUser.value.role)) {
        return list;
      }

      return list.filter((project: Project) => {
        if (project.id == DEFAULT_PROJECT_ID) {
          return true;
        }

        for (const member of project.memberList) {
          if (
            currentUser.value.id == member.principal.id &&
            props.allowedRoleList.includes(member.role)
          ) {
            return true;
          }
        }
        return false;
      });
    });

    const selectedIdNotInList = computed((): boolean => {
      if (props.selectedId == UNKNOWN_ID) {
        return false;
      }

      return (
        projectList.value.find((item) => {
          return item.id == props.selectedId;
        }) == null
      );
    });

    watch(
      () => props.selectedId,
      (cur) => {
        state.selectedId = cur;
      }
    );

    return {
      UNKNOWN_ID,
      DEFAULT_PROJECT_ID,
      state,
      projectList,
      selectedIdNotInList,
    };
  },
});
</script>
