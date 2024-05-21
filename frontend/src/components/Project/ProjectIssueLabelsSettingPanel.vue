<template>
  <div class="w-full flex flex-col justify-start items-start pt-6 space-y-4">
    <div>
      <h3 class="text-lg font-medium leading-7 text-main">
        {{ $t("project.settings.labels.issue-labels") }}
      </h3>
      <div class="flex space-x-2 items-center textinfolabel">
        {{ $t("project.settings.labels.force-issue-labels") }}
        <NSwitch
          class="ml-2"
          :size="'small'"
          :value="project.forceIssueLabels"
          :disabled="!allowEdit"
          @update:value="onSwitchUpdate"
        />
      </div>
    </div>
    <NDynamicTags
      :size="'large'"
      :disabled="!allowEdit"
      :value="labelValues"
      :render-tag="renderLabel"
      @update:value="onLabelsUpdate"
    />
    <div class="hidden">
      <NColorPicker
        :modes="['hex']"
        :show-alpha="false"
        :disabled="!allowEdit"
        :to="colorPickerLocator ?? false"
        :show="state.pendingEditColorIndex >= 0"
        :value="state.labels[state.pendingEditColorIndex]?.color"
        @update:value="onColorChange"
      />
    </div>
    <div v-if="valueChanged" class="w-full flex justify-end gap-x-3">
      <NButton :disabled="false" @click.prevent="revertChanges">
        {{ $t("common.revert") }}
      </NButton>
      <NButton type="primary" :disabled="!allowEdit" @click.prevent="onUpdate">
        {{ $t("common.update") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside } from "@vueuse/core";
import { isEqual, cloneDeep } from "lodash-es";
import { NDynamicTags, NTag, NColorPicker, NSwitch } from "naive-ui";
import { h, computed, nextTick, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useProjectV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import { Label } from "@/types/proto/v1/project_service";

interface LocalState {
  labels: Label[];
  pendingEditColorIndex: number;
}

const defaultColor = "#4f46e5";

const props = defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const state = reactive<LocalState>({
  labels: [...props.project.issueLabels],
  pendingEditColorIndex: -1,
});

const { t } = useI18n();
const projectStore = useProjectV1Store();

const colorPickerLocator = computed(() =>
  document.getElementById(`#color-${state.pendingEditColorIndex}`)
);

const labelValues = computed(() => state.labels.map((l) => l.value));

const valueChanged = computed(
  () => !isEqual(state.labels, props.project.issueLabels)
);

const onUpdate = async () => {
  const projectPatch = cloneDeep(props.project);
  projectPatch.issueLabels = state.labels;
  await projectStore.updateProject(projectPatch, ["issue_labels"]);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const revertChanges = () => {
  state.labels = [...props.project.issueLabels];
};

const onLabelsUpdate = (values: string[]) => {
  if (state.labels.length + 1 !== values.length) {
    return;
  }
  const newValue = values[values.length - 1];
  if (new Set(labelValues.value).has(newValue)) {
    return;
  }
  state.labels.push({
    color: defaultColor,
    value: newValue,
    group: "",
  });
};

const renderLabel = (value: string, index: number) => {
  const label = state.labels.find((l) => l.value === value) ?? {
    value,
    color: defaultColor,
  };

  return h(
    NTag,
    {
      closable: true,
      size: "large",
      disabled: !props.allowEdit,
      onClose: () => {
        state.labels.splice(index, 1);
      },
    },
    {
      default: () =>
        h("div", { class: "flex items-center gap-x-2" }, [
          h("div", {
            class: "w-4 h-4 rounded cursor-pointer relative",
            id: `#color-${index}`,
            style: `background-color: ${label.color};`,
            onClick: () => {
              state.pendingEditColorIndex = index;
              nextTick(() => {
                onClickOutside(colorPickerLocator, () => {
                  state.pendingEditColorIndex = -1;
                });
              });
            },
          }),
          label.value,
        ]),
    }
  );
};

const onColorChange = (color: string) => {
  if (state.pendingEditColorIndex < 0) {
    return;
  }
  state.labels[state.pendingEditColorIndex].color = color;
};

const onSwitchUpdate = async (on: boolean) => {
  const projectPatch = cloneDeep(props.project);
  projectPatch.forceIssueLabels = on;
  await projectStore.updateProject(projectPatch, ["force_issue_labels"]);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>

<style scoped lang="postcss">
:deep(.v-binder-follower-content) {
  @apply !translate-x-4 !translate-y-4;
}
</style>
