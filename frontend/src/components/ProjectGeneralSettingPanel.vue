<template>
  <form class="max-w-md space-y-4">
    <p class="text-lg font-medium leading-7 text-main">General</p>
    <div class="flex justify-between">
      <dl class="">
        <dt class="text-sm font-medium text-control-light">
          Name <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <input
            :disabled="!allowEdit"
            required
            autocomplete="off"
            id="projectName"
            type="text"
            class="textfield"
            v-model="state.name"
          />
        </dd>
      </dl>

      <dl class="">
        <dt class="text-sm font-medium text-control-light">
          Key <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <input
            :disabled="!allowEdit"
            required
            autocomplete="off"
            id="projectKey"
            type="text"
            class="textfield uppercase"
            v-model="state.key"
          />
        </dd>
      </dl>
    </div>

    <div v-if="allowEdit" class="flex justify-end">
      <button
        type="button"
        class="btn-primary"
        :disabled="!allowSave"
        @click.prevent="save"
      >
        Save
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, PropType, reactive } from "vue";
import { useStore } from "vuex";
import isEmpty from "lodash-es/isEmpty";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import ProjectMemberTable from "../components/ProjectMemberTable.vue";
import { Project, ProjectPatch } from "../types";
import { isProjectOwner } from "../utils";

interface LocalState {
  name: string;
  key: string;
}

export default {
  name: "ProjectGeneralSettingPanel",
  components: { PrincipalSelect, ProjectMemberTable },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const state = reactive<LocalState>({
      name: props.project.name,
      key: props.project.key,
    });

    // Only the project owner can edit the project info.
    // This means even the workspace owner won't be able to edit it.
    // There seems to be no good reason that workspace owner needs to mess up with the project name or key.
    const allowEdit = computed(() => {
      if (props.project.rowStatus == "ARCHIVED") {
        return false;
      }

      for (const member of props.project.memberList) {
        if (member.principal.id == currentUser.value.id) {
          if (isProjectOwner(member.role)) {
            return true;
          }
        }
      }
      return false;
    });

    const allowSave = computed((): boolean => {
      return (
        !isEmpty(state.name) &&
        !isEmpty(state.key) &&
        (state.name != props.project.name || state.key != props.project.key)
      );
    });

    const save = () => {
      const projectPatch: ProjectPatch = {
        updaterId: currentUser.value.id,
        name: state.name != props.project.name ? state.name : undefined,
        key: state.key != props.project.key ? state.key : undefined,
      };
      let subject = "project settings";
      if (state.name != props.project.name && state.key != props.project.key) {
        subject = "project name and key";
      } else if (state.name != props.project.name) {
        subject = "project name";
      } else if (state.key != props.project.key) {
        subject = "project key";
      }
      store
        .dispatch("project/patchProject", {
          projectId: props.project.id,
          projectPatch,
        })
        .then((updatedProject: Project) => {
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully updated ${subject}.`,
          });
          state.name = updatedProject.name;
          state.key = updatedProject.key;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    return {
      state,
      allowEdit,
      allowSave,
      save,
    };
  },
};
</script>
