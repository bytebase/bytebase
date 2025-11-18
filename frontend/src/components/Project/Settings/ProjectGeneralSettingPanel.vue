<template>
  <form class="w-full flex flex-col gap-y-4">
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
      <div class="font-medium">
        {{ $t("project.settings.project-labels.self") }}
      </div>
      <div class="text-sm text-gray-500 mb-3">
        {{ $t("project.settings.project-labels.description") }}
      </div>

      <LabelListEditor
        ref="labelListEditorRef"
        v-model:kv-list="labelKVList"
        :readonly="!allowEdit"
        :show-errors="true"
      />
    </div>
  </form>
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { NInput } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { LabelListEditor } from "@/components/Label";
import RequiredStar from "@/components/RequiredStar.vue";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  convertKVListToLabels,
  convertLabelsToKVList,
  extractProjectResourceName,
} from "@/utils";

interface LocalState {
  title: string;
}

const props = defineProps<{
  project: Project;
  allowEdit: boolean;
}>();

const projectV1Store = useProjectV1Store();
const labelListEditorRef = ref<InstanceType<typeof LabelListEditor>>();

const state = reactive<LocalState>({
  title: props.project.title,
});

// Convert labels to KVList format for LabelListEditor
const labelKVList = ref(
  convertLabelsToKVList(props.project.labels, true /* sort */)
);

// Watch for external changes to project labels
watch(
  () => props.project.labels,
  (newLabels) => {
    labelKVList.value = convertLabelsToKVList(newLabels, true /* sort */);
  }
);

const allowSave = computed((): boolean => {
  const titleChanged =
    props.project.name !== DEFAULT_PROJECT_NAME &&
    !isEmpty(state.title) &&
    state.title !== props.project.title;

  const labelsChanged = !isEqual(
    convertKVListToLabels(labelKVList.value, false),
    props.project.labels
  );

  // Check if there are validation errors
  const hasErrors = (labelListEditorRef.value?.flattenErrors ?? []).length > 0;

  return (titleChanged || labelsChanged) && !hasErrors;
});

const onUpdate = async () => {
  const projectPatch = cloneDeep(props.project);
  const updateMask: string[] = [];

  if (state.title !== props.project.title) {
    projectPatch.title = state.title;
    updateMask.push("title");
  }

  const currentLabels = convertKVListToLabels(labelKVList.value, false);
  if (!isEqual(currentLabels, props.project.labels)) {
    projectPatch.labels = currentLabels;
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
    labelKVList.value = convertLabelsToKVList(
      props.project.labels,
      true /* sort */
    );
  },
});
</script>
