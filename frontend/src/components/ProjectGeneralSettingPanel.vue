<template>
  <form class="max-w-md space-y-4">
    <p class="text-lg font-medium leading-7 text-main">
      {{ $t("common.general") }}
    </p>
    <div class="flex justify-between">
      <dl class="">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("common.name") }} <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <input
            id="projectName"
            v-model="state.name"
            :disabled="!allowEdit"
            required
            autocomplete="off"
            type="text"
            class="textfield"
          />
        </dd>
      </dl>

      <dl class="">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("common.key") }} <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <input
            id="projectKey"
            v-model="state.key"
            :disabled="!allowEdit"
            required
            autocomplete="off"
            type="text"
            class="textfield uppercase"
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
        {{ $t("common.save") }}
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { useStore } from "vuex";
import isEmpty from "lodash-es/isEmpty";
import { DEFAULT_PROJECT_ID, Project, ProjectPatch } from "../types";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

interface LocalState {
  name: string;
  key: string;
}

export default defineComponent({
  name: "ProjectGeneralSettingPanel",
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
    const { t } = useI18n();

    const state = reactive<LocalState>({
      name: props.project.name,
      key: props.project.key,
    });

    const allowSave = computed((): boolean => {
      return (
        props.project.id != DEFAULT_PROJECT_ID &&
        !isEmpty(state.name) &&
        !isEmpty(state.key) &&
        (state.name != props.project.name || state.key != props.project.key)
      );
    });

    const save = () => {
      const projectPatch: ProjectPatch = {
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
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("project.settings.success-updated-prompt", {
              subject: subject,
            }),
          });
          state.name = updatedProject.name;
          state.key = updatedProject.key;
        });
    };

    return {
      state,
      allowSave,
      save,
    };
  },
});
</script>
