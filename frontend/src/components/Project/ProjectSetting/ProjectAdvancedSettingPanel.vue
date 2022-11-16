<template>
  <div class="max-w-md space-y-4">
    <p class="text-lg font-medium leading-7 text-main">
      {{ $t("common.advanced") }}
    </p>
    <div class="space-y-4">
      <div class="flex items-center space-x-1">
        <label class="textlabel !font-bold">
          {{ $t("project.lgtm-check.self") }}
        </label>
        <FeatureBadge feature="bb.feature.lgtm" class="text-accent" />
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

      <label class="flex items-center space-x-4">
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
      </label>

      <label class="flex items-center space-x-4">
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
      </label>
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

    <FeatureModal
      v-if="state.missingFeature !== undefined"
      :feature="state.missingFeature"
      @cancel="state.missingFeature = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { FeatureType, LGTMCheckValue, Project, ProjectPatch } from "@/types";
import { featureToRef, pushNotification, useProjectStore } from "@/store";
import FeatureBadge from "@/components/FeatureBadge.vue";

interface LocalState {
  lgtmCheckValue: LGTMCheckValue;
  missingFeature: FeatureType | undefined;
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
const hasLGTMFeature = featureToRef("bb.feature.lgtm");

const state = reactive<LocalState>({
  lgtmCheckValue: props.project.lgtmCheckSetting.value,
  missingFeature: undefined,
});

const allowSave = computed((): boolean => {
  return state.lgtmCheckValue !== props.project.lgtmCheckSetting.value;
});

const save = () => {
  if (!hasLGTMFeature.value) {
    state.missingFeature = "bb.feature.lgtm";
    return;
  }

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
