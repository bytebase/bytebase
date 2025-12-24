<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-6">
    <div class="flex flex-col gap-y-2">
      <div class="font-medium">
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
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.forceIssueLabels"
          :text="true"
          :disabled="
            !allowUpdateIssueProjectSetting ||
            state.issueLabels.length === 0 ||
            loading
          "
        />
        <div class="textlabel flex items-center gap-x-2">
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
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t(
            "project.settings.issue-related.labels.force-issue-labels.description"
          )
        }}
      </div>
    </div>
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.enforceIssueTitle"
          :text="true"
          :disabled="!allowUpdateIssueProjectSetting || loading"
        />
        <span class="textlabel">
          {{ $t("project.settings.issue-related.enforce-issue-title.self") }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t("project.settings.issue-related.enforce-issue-title.description")
        }}
      </div>
    </div>
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.enforceSqlReview"
          :text="true"
          :disabled="!allowUpdateIssueProjectSetting || loading"
        />
        <span class="textlabel">
          {{ $t("project.settings.issue-related.enforce-sql-review.self") }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t("project.settings.issue-related.enforce-sql-review.description")
        }}
      </div>
    </div>
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.allowSelfApproval"
          :text="true"
          :disabled="!allowUpdateIssueProjectSetting || loading"
        />
        <span class="textlabel">
          {{ $t("project.settings.issue-related.allow-self-approval.self") }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t("project.settings.issue-related.allow-self-approval.description")
        }}
      </div>
    </div>
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.requireIssueApproval"
          :text="true"
          :disabled="!allowUpdateIssueProjectSetting || loading"
        />
        <span class="textlabel">
          {{
            $t("project.settings.issue-related.require-issue-approval.self")
          }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t(
            "project.settings.issue-related.require-issue-approval.description"
          )
        }}
      </div>
    </div>
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.requirePlanCheckNoError"
          :text="true"
          :disabled="!allowUpdateIssueProjectSetting || loading"
        />
        <span class="textlabel">
          {{
            $t(
              "project.settings.issue-related.require-plan-check-no-error.self"
            )
          }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t(
            "project.settings.issue-related.require-plan-check-no-error.description"
          )
        }}
      </div>
    </div>
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.autoEnableBackup"
          :text="true"
          :disabled="!allowUpdateIssueProjectSetting || loading"
        />
        <span class="textlabel">
          {{ $t("project.settings.issue-related.auto-enable-backup.self") }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t("project.settings.issue-related.auto-enable-backup.description")
        }}
      </div>
    </div>
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.skipBackupErrors"
          :text="true"
          :disabled="!allowUpdateIssueProjectSetting || loading"
        />
        <span class="textlabel">
          {{ $t("project.settings.issue-related.skip-backup-errors.self") }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t("project.settings.issue-related.skip-backup-errors.description")
        }}
      </div>
    </div>
    <div>
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.postgresDatabaseTenantMode"
          :text="true"
          :disabled="!allowUpdateIssueProjectSetting || loading"
        />
        <span class="textlabel">
          {{
            $t(
              "project.settings.issue-related.postgres-database-tenant-mode.self"
            )
          }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t(
            "project.settings.issue-related.postgres-database-tenant-mode.description"
          )
        }}
      </div>
    </div>
    <div>
      <p class="">
        <span class="textlabel">
          {{ $t("project.settings.issue-related.max-retries.self") }}
        </span>
      </p>
      <p class="mb-3 text-sm text-gray-400">
        {{ $t("project.settings.issue-related.max-retries.description") }}
      </p>
      <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
        <NInputNumber
          :value="state.executionRetryPolicy?.maximumRetries ?? 0"
          :disabled="!allowUpdateIssueProjectSetting || loading"
          class="w-60"
          :min="0"
          :precision="0"
          @update:value="handleInput"
        >
          <template #suffix> Times </template>
        </NInputNumber>
      </div>
    </div>
    <div>
      <p class="">
        <span class="textlabel">
          {{ $t("project.settings.issue-related.ci-sampling-size.self") }}
        </span>
      </p>
      <p class="mb-3 text-sm text-gray-400">
        {{
          $t("project.settings.issue-related.ci-sampling-size.description")
        }}
      </p>
      <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
        <NInputNumber
          :value="state.ciSamplingSize"
          :disabled="!allowUpdateIssueProjectSetting || loading"
          class="w-60"
          :min="0"
          :precision="0"
          @update:value="handleCiSamplingSizeInput"
        >
        </NInputNumber>
      </div>
    </div>
    <div>
      <p class="">
        <span class="textlabel">
          {{
            $t(
              "project.settings.issue-related.parallel_tasks_per_rollout.self"
            )
          }}
        </span>
      </p>
      <p class="mb-3 text-sm text-gray-400">
        {{
          $t(
            "project.settings.issue-related.parallel_tasks_per_rollout.description"
          )
        }}
      </p>
      <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
        <NInputNumber
          :value="state.parallelTasksPerRollout"
          :disabled="!allowUpdateIssueProjectSetting || loading"
          class="w-60"
          :min="0"
          :precision="0"
          @update:value="handleParallelTasksPerRolloutInput"
        >
        </NInputNumber>
      </div>
    </div>
  </div>
</template>

<script setup lang="tsx">
import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import { TriangleAlertIcon } from "lucide-vue-next";
import {
  NColorPicker,
  NDynamicTags,
  NInputNumber,
  NTag,
  NTooltip,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { Switch } from "@/components/v2";
import { useProjectV1Store } from "@/store";
import type {
  Label,
  Project,
  Project_ExecutionRetryPolicy,
} from "@/types/proto-es/v1/project_service_pb";
import {
  LabelSchema,
  Project_ExecutionRetryPolicySchema,
} from "@/types/proto-es/v1/project_service_pb";

interface LocalState {
  issueLabels: Label[];
  enforceSqlReview: boolean;
  forceIssueLabels: boolean;
  enforceIssueTitle: boolean;
  allowSelfApproval: boolean;
  autoEnableBackup: boolean;
  skipBackupErrors: boolean;
  postgresDatabaseTenantMode: boolean;
  executionRetryPolicy: Project_ExecutionRetryPolicy | undefined;
  ciSamplingSize: number;
  parallelTasksPerRollout: number;
  requireIssueApproval: boolean;
  requirePlanCheckNoError: boolean;
}

const getInitialLocalState = (): LocalState => {
  if (!props.project) {
    return {
      issueLabels: [],
      forceIssueLabels: false,
      enforceIssueTitle: false,
      enforceSqlReview: false,
      autoEnableBackup: false,
      skipBackupErrors: false,
      postgresDatabaseTenantMode: false,
      allowSelfApproval: false,
      executionRetryPolicy: createProto(Project_ExecutionRetryPolicySchema, {
        maximumRetries: 0,
      }),
      ciSamplingSize: 0,
      parallelTasksPerRollout: 1,
      requireIssueApproval: false,
      requirePlanCheckNoError: false,
    };
  }
  return {
    issueLabels: cloneDeep(props.project.issueLabels),
    forceIssueLabels: props.project.forceIssueLabels,
    enforceIssueTitle: props.project.enforceIssueTitle,
    enforceSqlReview: props.project.enforceSqlReview,
    autoEnableBackup: props.project.autoEnableBackup,
    skipBackupErrors: props.project.skipBackupErrors,
    postgresDatabaseTenantMode: props.project.postgresDatabaseTenantMode,
    allowSelfApproval: props.project.allowSelfApproval,
    executionRetryPolicy: props.project.executionRetryPolicy
      ? createProto(Project_ExecutionRetryPolicySchema, {
          maximumRetries: props.project.executionRetryPolicy.maximumRetries,
        })
      : createProto(Project_ExecutionRetryPolicySchema, {
          maximumRetries: 0,
        }),
    ciSamplingSize: props.project.ciSamplingSize,
    parallelTasksPerRollout: props.project.parallelTasksPerRollout,
    requireIssueApproval: props.project.requireIssueApproval,
    requirePlanCheckNoError: props.project.requirePlanCheckNoError,
  };
};

const defaultColor = "#4f46e5";

const props = defineProps<{
  project: Project;
  allowEdit: boolean;
}>();

const projectStore = useProjectV1Store();
const state = reactive<LocalState>(getInitialLocalState());
const loading = ref<boolean>(false);

const labelValues = computed(() => state.issueLabels.map((l) => l.value));

const valueChanged = computed(() => !isEqual(state, getInitialLocalState()));

const allowUpdateIssueProjectSetting = computed(() => {
  return props.allowEdit;
});

const onLabelsUpdate = (values: string[]) => {
  if (state.issueLabels.length + 1 !== values.length) {
    return;
  }
  const newValue = values[values.length - 1];
  if (new Set(labelValues.value).has(newValue)) {
    return;
  }
  state.issueLabels.push(
    createProto(LabelSchema, {
      color: defaultColor,
      value: newValue,
      group: "",
    })
  );
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
            class="w-full! h-full!"
            modes={["hex"]}
            showAlpha={false}
            value={label.color}
            renderLabel={() => (
              <div
                class="w-4 h-4 rounded-sm cursor-pointer relative"
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
  if (state.enforceSqlReview !== props.project.enforceSqlReview) {
    mask.push("enforce_sql_review");
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
  if (
    !isEqual(
      state.executionRetryPolicy?.maximumRetries ?? 0,
      props.project.executionRetryPolicy?.maximumRetries ?? 0
    )
  ) {
    mask.push("execution_retry_policy");
  }
  if (!isEqual(state.ciSamplingSize, props.project.ciSamplingSize || 0)) {
    mask.push("ci_sampling_size");
  }
  if (
    !isEqual(
      state.parallelTasksPerRollout,
      props.project.parallelTasksPerRollout || 0
    )
  ) {
    mask.push("parallel_tasks_per_rollout");
  }
  if (state.requireIssueApproval !== props.project.requireIssueApproval) {
    mask.push("require_issue_approval");
  }
  if (state.requirePlanCheckNoError !== props.project.requirePlanCheckNoError) {
    mask.push("require_plan_check_no_error");
  }
  return mask;
});

const handleInput = (value: number | null) => {
  if (value === null) return;
  if (value === undefined) return;
  state.executionRetryPolicy = createProto(Project_ExecutionRetryPolicySchema, {
    maximumRetries: value,
  });
};

const handleCiSamplingSizeInput = (value: number | null) => {
  if (value === null || value === undefined) return;
  state.ciSamplingSize = value;
};

const handleParallelTasksPerRolloutInput = (value: number | null) => {
  if (value === null || value === undefined) return;
  state.parallelTasksPerRollout = value;
};

defineExpose({
  isDirty: valueChanged,
  update: doUpdate,
  revert: () => {
    Object.assign(state, getInitialLocalState());
  },
});
</script>
