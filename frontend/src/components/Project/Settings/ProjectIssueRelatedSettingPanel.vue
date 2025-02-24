<template>
  <div class="w-full flex flex-col justify-start items-start pt-6 space-y-4">
    <div class="space-y-2 mb-4">
      <div class="text-lg font-medium text-main">
        {{ $t("project.settings.issue-related.labels.self") }}
        <div class="textinfolabel">
          {{ $t("project.settings.issue-related.labels.description") }}
        </div>
      </div>
      <NDynamicTags
        :size="'large'"
        :disabled="!allowEdit || loading"
        :value="labelValues"
        :render-tag="renderLabel"
        :input-props="{
          placeholder: $t('project.settings.issue-related.labels.placeholder'),
          clearable: true,
        }"
        :input-style="'min-width: 12rem;'"
        @update:value="onLabelsUpdate"
      />
    </div>
    <h3 class="flex flex-row items-center gap-2">
      <span class="text-lg font-medium text-main">{{
        $t("project.settings.issue-related.self")
      }}</span>
      <FeatureBadge feature="bb.feature.issue-project-setting" />
    </h3>
    <div class="w-full flex flex-col justify-start items-start gap-2">
      <div>
        <NCheckbox
          v-model:checked="state.allowModifyStatement"
          size="large"
          :disabled="!allowUpdateIssueProjectSetting || loading"
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
          :disabled="!allowUpdateIssueProjectSetting || loading"
          :label="$t('project.settings.issue-related.auto-resolve-issue.self')"
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t("project.settings.issue-related.auto-resolve-issue.description")
          }}
        </p>
      </div>
      <div>
        <NCheckbox
          v-model:checked="state.forceIssueLabels"
          size="large"
          :disabled="
            !allowUpdateIssueProjectSetting ||
            state.issueLabels.length === 0 ||
            loading
          "
        >
          <template #default>
            <div class="flex items-center gap-x-2">
              {{
                $t(
                  "project.settings.issue-related.labels.force-issue-labels.self"
                )
              }}
              <NTooltip v-if="allowEdit && state.issueLabels.length === 0">
                <template #trigger>
                  <TriangleAlertIcon class="w-4 text-warning" />
                </template>
                {{
                  $t(
                    "project.settings.issue-related.labels.force-issue-labels.warning"
                  )
                }}
              </NTooltip>
            </div>
          </template>
        </NCheckbox>
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
          v-model:checked="state.enforceIssueTitle"
          size="large"
          :disabled="!allowUpdateIssueProjectSetting || loading"
          :label="$t('project.settings.issue-related.enforce-issue-title.self')"
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t("project.settings.issue-related.enforce-issue-title.description")
          }}
        </p>
      </div>
      <div>
        <NCheckbox
          v-model:checked="state.allowSelfApproval"
          size="large"
          :disabled="!allowUpdateIssueProjectSetting || loading"
          :label="$t('project.settings.issue-related.allow-self-approval.self')"
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t("project.settings.issue-related.allow-self-approval.description")
          }}
        </p>
      </div>
      <div>
        <NCheckbox
          v-model:checked="state.autoEnableBackup"
          size="large"
          :disabled="!allowUpdateIssueProjectSetting || loading"
          :label="$t('project.settings.issue-related.auto-enable-backup.self')"
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t("project.settings.issue-related.auto-enable-backup.description")
          }}
        </p>
      </div>
      <div>
        <NCheckbox
          v-model:checked="state.skipBackupErrors"
          size="large"
          :disabled="!allowUpdateIssueProjectSetting || loading"
          :label="$t('project.settings.issue-related.skip-backup-errors.self')"
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t("project.settings.issue-related.skip-backup-errors.description")
          }}
        </p>
      </div>
      <div>
        <NCheckbox
          v-model:checked="state.postgresDatabaseTenantMode"
          size="large"
          :disabled="!allowUpdateIssueProjectSetting || loading"
          :label="
            $t(
              'project.settings.issue-related.postgres-database-tenant-mode.self'
            )
          "
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{
            $t(
              "project.settings.issue-related.postgres-database-tenant-mode.description"
            )
          }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="tsx">
import { isEqual, cloneDeep } from "lodash-es";
import { TriangleAlertIcon } from "lucide-vue-next";
import {
  NDynamicTags,
  NTag,
  NColorPicker,
  NCheckbox,
  NTooltip,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { FeatureBadge } from "@/components/FeatureGuard";
import { hasFeature, useProjectV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import { Label } from "@/types/proto/v1/project_service";

interface LocalState {
  issueLabels: Label[];
  allowModifyStatement: boolean;
  autoResolveIssue: boolean;
  forceIssueLabels: boolean;
  enforceIssueTitle: boolean;
  allowSelfApproval: boolean;
  autoEnableBackup: boolean;
  skipBackupErrors: boolean;
  postgresDatabaseTenantMode: boolean;
}

const getInitialLocalState = (): LocalState => {
  const project = props.project;
  return {
    issueLabels: [...cloneDeep(project.issueLabels)],
    allowModifyStatement: project.allowModifyStatement,
    autoResolveIssue: project.autoResolveIssue,
    forceIssueLabels: project.forceIssueLabels,
    enforceIssueTitle: project.enforceIssueTitle,
    allowSelfApproval: project.allowSelfApproval,
    autoEnableBackup: project.autoEnableBackup,
    skipBackupErrors: project.skipBackupErrors,
    postgresDatabaseTenantMode: project.postgresDatabaseTenantMode,
  };
};

const defaultColor = "#4f46e5";

const props = defineProps<{
  project: ComposedProject;
  allowEdit: boolean;
}>();

const projectStore = useProjectV1Store();
const state = reactive<LocalState>(getInitialLocalState());
const loading = ref<boolean>(false);

const labelValues = computed(() => state.issueLabels.map((l) => l.value));

const valueChanged = computed(() => !isEqual(state, getInitialLocalState()));

const allowUpdateIssueProjectSetting = computed(() => {
  return props.allowEdit && hasFeature("bb.feature.issue-project-setting");
});

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
  if (loading.value) {
    return;
  }
  if (updateMask.value.length === 0) {
    return;
  }
  loading.value = true;
  const projectPatch = {
    ...cloneDeep(props.project),
    ...state,
  };
  try {
    await projectStore.updateProject(projectPatch, updateMask.value);
  } finally {
    loading.value = false;
  }
};

const updateMask = computed(() => {
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
  if (state.enforceIssueTitle !== props.project.enforceIssueTitle) {
    mask.push("enforce_issue_title");
  }
  if (state.allowSelfApproval !== props.project.allowSelfApproval) {
    mask.push("allow_self_approval");
  }
  if (!isEqual(state.autoEnableBackup, props.project.autoEnableBackup)) {
    mask.push("auto_enable_backup");
  }
  if (!isEqual(state.skipBackupErrors, props.project.skipBackupErrors)) {
    mask.push("skip_backup_errors");
  }
  if (
    !isEqual(
      state.postgresDatabaseTenantMode,
      props.project.postgresDatabaseTenantMode
    )
  ) {
    mask.push("postgres_database_tenant_mode");
  }
  return mask;
});

defineExpose({
  isDirty: valueChanged,
  update: doUpdate,
  revert: () => {
    Object.assign(state, getInitialLocalState());
  },
});
</script>
