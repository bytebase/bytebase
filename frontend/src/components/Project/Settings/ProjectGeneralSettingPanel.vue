<template>
  <form class="w-full space-y-4">
    <div>
      <div class="font-medium">
        {{ $t("common.name") }}
        <RequiredStar />
      </div>
      <NInput
        id="projectName"
        class="mt-1"
        v-model:value="state.title"
        :disabled="!allowEdit"
        required
      />
      <div class="mt-1">
        <ResourceIdField
          resource-type="project"
          :value="extractProjectResourceName(project.name)"
          :readonly="true"
        />
      </div>
    </div>

    <!-- Project Labels Section -->
    <div>
      <div class="font-medium mb-2">
        {{ $t("project.settings.project-labels.self") }}
      </div>
      <div class="text-sm text-gray-500 mb-3">
        {{ $t("project.settings.project-labels.description") }}
      </div>

      <div class="space-y-2">
        <!-- Existing Labels -->
        <div
          v-for="(value, key) in sortedLabels"
          :key="key"
          class="flex items-center gap-2"
        >
          <NInput
            :value="key"
            :disabled="true"
            class="flex-1"
            :placeholder="$t('project.settings.project-labels.key')"
          />
          <span class="text-gray-500">:</span>
          <NInput
            v-model:value="state.labels[key]"
            :disabled="!allowEdit"
            class="flex-1"
            :placeholder="$t('project.settings.project-labels.value')"
            @blur="validateLabel(key, state.labels[key])"
          />
          <NButton v-if="allowEdit" tertiary circle @click="removeLabel(key)">
            <template #icon>
              <heroicons:x-mark class="w-4 h-4" />
            </template>
          </NButton>
        </div>

        <!-- Add New Label -->
        <div
          v-if="allowEdit && Object.keys(state.labels).length < 64"
          class="flex items-center gap-2"
        >
          <NInput
            v-model:value="newLabel.key"
            class="flex-1"
            :placeholder="$t('project.settings.project-labels.key-placeholder')"
            @keyup.enter="addLabel"
          />
          <span class="text-gray-500">:</span>
          <NInput
            v-model:value="newLabel.value"
            class="flex-1"
            :placeholder="
              $t('project.settings.project-labels.value-placeholder')
            "
            @keyup.enter="addLabel"
          />
          <NButton secondary :disabled="!canAddLabel" @click="addLabel">
            {{ $t("project.settings.project-labels.add-label") }}
          </NButton>
        </div>

        <!-- Validation Error for New Label -->
        <div v-if="validateNewLabel" class="text-sm text-red-600 mt-1">
          {{ validateNewLabel }}
        </div>

        <!-- Validation Error for Existing Labels -->
        <div v-if="labelError" class="text-sm text-red-600 mt-1">
          {{ labelError }}
        </div>
      </div>
    </div>
  </form>
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { NInput, NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import RequiredStar from "@/components/RequiredStar.vue";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName } from "@/utils";

interface LocalState {
  title: string;
  labels: Record<string, string>;
}

const props = defineProps<{
  project: Project;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();

const state = reactive<LocalState>({
  title: props.project.title,
  labels: { ...props.project.labels },
});

const newLabel = reactive({
  key: "",
  value: "",
});

const labelError = ref("");
const newLabelError = ref("");

const allowSave = computed((): boolean => {
  return (
    (props.project.name !== DEFAULT_PROJECT_NAME &&
      !isEmpty(state.title) &&
      state.title !== props.project.title) ||
    !isEqual(state.labels, props.project.labels)
  );
});

const canAddLabel = computed((): boolean => {
  return (
    newLabel.key.trim() !== "" &&
    validateLabelKey(newLabel.key) &&
    validateLabelValue(newLabel.value) &&
    !(newLabel.key in state.labels)
  );
});

// Real-time validation for new label inputs
const validateNewLabel = computed((): string => {
  if (!newLabel.key.trim()) {
    return "";
  }

  // Check key format
  if (!validateLabelKey(newLabel.key)) {
    return t("project.settings.project-labels.validation.invalid-key");
  }

  // Check duplicate
  if (newLabel.key in state.labels) {
    return t("project.settings.project-labels.validation.duplicate-key");
  }

  // Check value format if value is provided
  if (newLabel.value && !validateLabelValue(newLabel.value)) {
    return t("project.settings.project-labels.validation.invalid-value");
  }

  // Check max labels
  if (Object.keys(state.labels).length >= 64) {
    return t("project.settings.project-labels.validation.max-labels");
  }

  return "";
});

// Sort labels alphabetically by key for consistent display order
// (JSONB doesn't preserve insertion order)
const sortedLabels = computed(() => {
  return Object.keys(state.labels)
    .sort()
    .reduce((acc, key) => {
      acc[key] = state.labels[key];
      return acc;
    }, {} as Record<string, string>);
});

const validateLabelKey = (key: string): boolean => {
  const keyPattern = /^[a-z][a-z0-9_-]{0,62}$/;
  return keyPattern.test(key);
};

const validateLabelValue = (value: string): boolean => {
  const valuePattern = /^[a-zA-Z0-9_-]{0,63}$/;
  return valuePattern.test(value);
};

const validateLabel = (key: string, value: string): void => {
  labelError.value = "";
  if (!validateLabelKey(key)) {
    labelError.value = t(
      "project.settings.project-labels.validation.invalid-key"
    );
    return;
  }
  if (!validateLabelValue(value)) {
    labelError.value = t(
      "project.settings.project-labels.validation.invalid-value"
    );
    return;
  }
};

const addLabel = (): void => {
  if (!canAddLabel.value) return;

  // Add label
  state.labels[newLabel.key] = newLabel.value;
  newLabel.key = "";
  newLabel.value = "";
  newLabelError.value = "";
};

const removeLabel = (key: string): void => {
  delete state.labels[key];
};

const onUpdate = async () => {
  const projectPatch = cloneDeep(props.project);
  const updateMask: string[] = [];

  if (state.title !== props.project.title) {
    projectPatch.title = state.title;
    updateMask.push("title");
  }

  if (!isEqual(state.labels, props.project.labels)) {
    projectPatch.labels = state.labels;
    updateMask.push("labels");
  }

  if (updateMask.length > 0) {
    await projectV1Store.updateProject(projectPatch, updateMask);
  }
};

defineExpose({
  isDirty: allowSave,
  update: onUpdate,
  revert: () => {
    state.title = props.project.title;
    state.labels = { ...props.project.labels };
    newLabel.key = "";
    newLabel.value = "";
    labelError.value = "";
    newLabelError.value = "";
  },
});
</script>
