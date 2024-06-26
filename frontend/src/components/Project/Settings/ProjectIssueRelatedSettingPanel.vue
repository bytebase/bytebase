<template>
  <div class="w-full flex flex-col justify-start items-start pt-6 space-y-4">
    <h3 class="text-lg font-medium text-main">
      {{ $t("project.settings.issue-related.self") }}
    </h3>
    <div class="w-full flex flex-col justify-start items-start gap-2">
      <span class="textlabel">{{
        $t("project.settings.issue-related.labels.self")
      }}</span>
      <NDynamicTags
        :size="'large'"
        :disabled="!allowEdit"
        :value="labelValues"
        :render-tag="renderLabel"
        @update:value="onLabelsUpdate"
      />
      <div>
        <NCheckbox
          v-model:checked="state.forceIssueLabels"
          size="large"
          :disabled="!allowEdit || state.issueLabels.length === 0"
          :label="
            $t('project.settings.issue-related.labels.force-issue-labels.self')
          "
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t(
              "project.settings.issue-related.labels.force-issue-labels.description"
            )
          }}
        </p>
      </div>
      <div>
        <NCheckbox
          v-model:checked="state.allowModifyStatement"
          size="large"
          :disabled="!allowEdit"
          :label="
            $t('project.settings.issue-related.allow-modify-statement.self')
          "
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t(
              "project.settings.issue-related.allow-modify-statement.description"
            )
          }}
        </p>
      </div>
      <div>
        <NCheckbox
          v-model:checked="state.autoResolveIssue"
          size="large"
          :disabled="!allowEdit"
          :label="$t('project.settings.issue-related.auto-resolve-issue.self')"
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t("project.settings.issue-related.auto-resolve-issue.description")
          }}
        </p>
      </div>
    </div>
    <div class="w-full flex justify-end gap-x-3">
      <NButton
        type="primary"
        :disabled="!valueChanged || !allowEdit"
        @click.prevent="doUpdate"
      >
        {{ $t("common.update") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="tsx">
import { isEqual, cloneDeep } from "lodash-es";
import { NButton, NDynamicTags, NTag, NColorPicker, NCheckbox } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useProjectV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import { Label } from "@/types/proto/v1/project_service";

const getInitialLocalState = (): LocalState => {
  const project = props.project;
  return {
    issueLabels: [...cloneDeep(project.issueLabels)],
    forceIssueLabels: project.forceIssueLabels,
    allowModifyStatement: project.allowModifyStatement,
    autoResolveIssue: project.autoResolveIssue,
  };
};

interface LocalState {
  issueLabels: Label[];
  forceIssueLabels: boolean;
  allowModifyStatement: boolean;
  autoResolveIssue: boolean;
}

const defaultColor = "#4f46e5";

const props = defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const state = reactive<LocalState>(getInitialLocalState());

const { t } = useI18n();
const projectStore = useProjectV1Store();

const labelValues = computed(() => state.issueLabels.map((l) => l.value));

const valueChanged = computed(() => !isEqual(state, getInitialLocalState()));

const onLabelsUpdate = (values: string[]) => {
  if (state.issueLabels.length + 1 !== values.length) {
    return;
  }
  const newValue = values[values.length - 1];
  if (new Set(labelValues.value).has(newValue)) {
    return;
  }
  state.issueLabels.push({
    color: defaultColor,
    value: newValue,
    group: "",
  });
};

const renderLabel = (value: string, index: number) => {
  const label = state.issueLabels.find((l) => l.value === value) ?? {
    value,
    color: defaultColor,
  };

  return (
    <NTag
      size="large"
      closable
      disabled={!props.allowEdit}
      onClose={() => {
        state.issueLabels.splice(index, 1);
        if (state.issueLabels.length === 0) {
          state.forceIssueLabels = false;
        }
      }}
    >
      <div class="flex flex-row items-start justify-center gap-x-2">
        <div class="w-4 h-4 relative">
          <NColorPicker
            class="!w-full !h-full"
            modes={["hex"]}
            showAlpha={false}
            value={label.color}
            renderLabel={() => (
              <div
                class="w-4 h-4 rounded cursor-pointer relative"
                style={{ backgroundColor: label.color }}
              ></div>
            )}
            onUpdateValue={(color: string) => (label.color = color)}
          />
        </div>
        <span>{label.value}</span>
      </div>
    </NTag>
  );
};

const doUpdate = async () => {
  const projectPatch = {
    ...cloneDeep(props.project),
    ...state,
  };
  await projectStore.updateProject(projectPatch, getUpdateMask());
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const getUpdateMask = () => {
  const mask: string[] = [];
  if (!isEqual(state.issueLabels, props.project.issueLabels)) {
    mask.push("issue_labels");
  }
  if (state.forceIssueLabels !== props.project.forceIssueLabels) {
    mask.push("force_issue_labels");
  }
  if (state.allowModifyStatement !== props.project.allowModifyStatement) {
    mask.push("allow_modify_statement");
  }
  if (state.autoResolveIssue !== props.project.autoResolveIssue) {
    mask.push("auto_resolve_issue");
  }
  return mask;
};
</script>
