<template>
  <div class="max-w-md space-y-4">
    <p class="text-lg font-medium leading-7 text-main">
      {{ $t("common.advanced") }}
    </p>
    <div class="space-y-4">
      <div class="flex items-center">
        <label class="textlabel">
          {{ $t("project.lgtm-check.self") }}
        </label>
      </div>

      <label class="flex items-center space-x-4">
        <input
          v-model="state.lgtmCheckValue"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="DISABLED"
          :disabled="!allowEdit"
        />
        <div class="-mt-0.5">
          <div class="textlabel">
            {{ $t("project.lgtm-check.disabled") }}
          </div>
        </div>
      </label>

      <div class="flex items-center space-x-4">
        <input
          v-model="state.lgtmCheckValue"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="PROJECT_MEMBER"
          :disabled="!allowEdit"
        />
        <div class="-mt-0.5">
          <div class="textlabel">
            {{ $t("project.lgtm-check.project-member") }}
          </div>
        </div>
      </div>

      <div class="flex items-center space-x-4">
        <input
          v-model="state.lgtmCheckValue"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="PROJECT_OWNER"
          :disabled="!allowEdit"
        />
        <div class="-mt-0.5">
          <div class="textlabel">
            {{ $t("project.lgtm-check.project-owner") }}
          </div>
        </div>
      </div>
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
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { LGTMCheckValue, Project, ProjectPatch } from "@/types";
import { pushNotification, useProjectStore } from "@/store";

interface LocalState {
  lgtmCheckValue: LGTMCheckValue;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  allowEdit: {
    default: true,
    type: Boolean,
  },
});
const { t } = useI18n();
const projectStore = useProjectStore();

const state = reactive<LocalState>({
  lgtmCheckValue: props.project.lgtmCheckSetting.value,
});

const allowSave = computed((): boolean => {
  return state.lgtmCheckValue !== props.project.lgtmCheckSetting.value;
});

const save = () => {
  const projectPatch: ProjectPatch = {
    lgtmCheckSetting: {
      value: state.lgtmCheckValue,
    },
  };

  projectStore
    .patchProject({
      projectId: props.project.id,
      projectPatch,
    })
    .then((updatedProject: Project) => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.settings.success-updated"),
      });
    });
};

// Sync the local state if props changed
watch(
  () => props.project.lgtmCheckSetting.value,
  (value) => {
    state.lgtmCheckValue = value;
  }
);
</script>
