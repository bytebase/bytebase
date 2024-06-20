<template>
  <form class="w-full space-y-4 mx-auto">
    <p class="text-lg font-medium leading-7 text-main">
      {{ $t("common.general") }}
    </p>
    <div class="flex justify-start items-start gap-6">
      <dl class="">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("common.name") }} <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <NInput
            id="projectName"
            v-model:value="state.title"
            :disabled="!allowEdit"
            required
          />
        </dd>
        <div class="mt-1">
          <ResourceIdField
            resource-type="project"
            :value="extractProjectResourceName(project.name)"
            :readonly="true"
          />
        </div>
      </dl>

      <dl class="">
        <dt class="flex text-sm font-medium text-control-light">
          {{ $t("common.key") }}
          <NTooltip>
            <template #trigger>
              <heroicons-outline:information-circle class="ml-1 w-4 h-auto" />
            </template>
            {{ $t("project.key-hint") }}
          </NTooltip>
          <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <NInput
            id="projectKey"
            v-model:value="state.key"
            :disabled="!allowEdit"
            required
            @update:value="(val: string) => (state.key = val.toUpperCase())"
          />
        </dd>
      </dl>
    </div>

    <div class="flex flex-col">
      <div for="name" class="text-sm font-medium text-control-light">
        {{ $t("common.mode") }}
        <span class="text-red-600">*</span>
      </div>
      <div class="mt-2 textlabel">
        <ProjectModeRadioGroup
          v-model:value="state.tenantMode"
          :disabled="!allowEdit"
        />
      </div>
    </div>

    <div class="flex flex-col">
      <div for="name" class="text-sm font-medium text-control-light">
        {{ $t("common.issue") }}
      </div>
      <div class="mt-2">
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <label
              class="flex items-center gap-x-2"
              :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
            >
              <NCheckbox
                :disabled="!allowEdit"
                :checked="restrictIssueCreationForSQLReview"
                :label="
                  $t(
                    'settings.general.workspace.restrict-issue-creation-for-sql-review.title'
                  )
                "
                @update:checked="handleRestrictIssueCreationForSQLReviewToggle"
              />
            </label>
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{ $t("settings.general.workspace.only-admin-can-edit") }}
          </span>
        </NTooltip>
        <div class="mb-3 text-sm text-gray-400">
          {{
            $t(
              "settings.general.workspace.restrict-issue-creation-for-sql-review.description"
            )
          }}
        </div>
      </div>
    </div>

    <div v-if="allowEdit" class="flex justify-end">
      <NButton type="primary" :disabled="!allowSave" @click.prevent="save">
        {{ $t("common.update") }}
      </NButton>
    </div>

    <FeatureModal
      :open="!!state.requiredFeature"
      :feature="state.requiredFeature"
      @cancel="state.requiredFeature = undefined"
    />
  </form>
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty } from "lodash-es";
import { NTooltip, NCheckbox } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import {
  hasFeature,
  pushNotification,
  useProjectV1Store,
  usePolicyV1Store,
} from "@/store";
import type { FeatureType } from "@/types";
import { DEFAULT_PROJECT_ID } from "@/types";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import type { Project } from "@/types/proto/v1/project_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import { extractProjectResourceName } from "@/utils";

interface LocalState {
  title: string;
  key: string;
  tenantMode: TenantMode;
  requiredFeature: FeatureType | undefined;
}

const props = defineProps<{
  project: Project;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();
const policyV1Store = usePolicyV1Store();

watchEffect(async () => {
  await policyV1Store.getOrFetchPolicyByParentAndType({
    parentPath: props.project.name,
    policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
  });
});

const state = reactive<LocalState>({
  title: props.project.title,
  key: props.project.key,
  tenantMode: props.project.tenantMode,
  requiredFeature: undefined,
});

const allowSave = computed((): boolean => {
  return (
    parseInt(props.project.uid, 10) !== DEFAULT_PROJECT_ID &&
    !isEmpty(state.title) &&
    (state.title !== props.project.title ||
      state.key !== props.project.key ||
      state.tenantMode !== props.project.tenantMode)
  );
});

const restrictIssueCreationForSQLReview = computed((): boolean => {
  return (
    policyV1Store.getPolicyByParentAndType({
      parentPath: props.project.name,
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    })?.restrictIssueCreationForSqlReviewPolicy?.disallow ?? false
  );
});

const handleRestrictIssueCreationForSQLReviewToggle = async (on: boolean) => {
  await policyV1Store.createPolicy(props.project.name, {
    type: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    resourceType: PolicyResourceType.PROJECT,
    restrictIssueCreationForSqlReviewPolicy: {
      disallow: on,
    },
  });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};

const save = () => {
  const projectPatch = cloneDeep(props.project);
  const updateMask: string[] = [];
  if (state.title !== props.project.title) {
    projectPatch.title = state.title;
    updateMask.push("title");
  }
  if (state.key !== props.project.key) {
    projectPatch.key = state.key;
    updateMask.push("key");
  }
  if (state.tenantMode !== props.project.tenantMode) {
    if (state.tenantMode === TenantMode.TENANT_MODE_ENABLED) {
      if (!hasFeature("bb.feature.multi-tenancy")) {
        state.tenantMode = TenantMode.TENANT_MODE_DISABLED;
        state.requiredFeature = "bb.feature.multi-tenancy";
        return;
      }
    }
    projectPatch.tenantMode = state.tenantMode;
    updateMask.push("tenant_mode");
  }
  projectV1Store.updateProject(projectPatch, updateMask).then((updated) => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.settings.success-updated"),
    });
  });
};
</script>
