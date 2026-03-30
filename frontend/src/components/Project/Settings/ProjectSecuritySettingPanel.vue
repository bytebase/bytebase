<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-6">
    <ComponentPermissionGuard
      :project="project"
      :permissions="['bb.policies.get']"
    >
      <div class="w-full flex flex-col justify-start items-start gap-y-6">
        <SQLReviewForResource
          ref="sqlReviewForResourceRef"
          :resource="project.name"
        />

        <!-- Maximum SQL Result Rows (project-level) -->
        <div>
          <p class="font-medium flex flex-row justify-start items-center">
            <span class="mr-2">
              {{ $t("settings.general.workspace.maximum-sql-result.rows.self") }}
            </span>
            <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
          </p>
          <p class="text-sm text-gray-400 mt-1">
            {{
              $t("settings.general.workspace.maximum-sql-result.rows.description")
            }}
            <span class="font-semibold! textinfolabel">
              {{ $t("settings.general.workspace.no-limit") }}
            </span>
          </p>
          <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
            <PermissionGuardWrapper
              v-slot="slotProps"
              :project="project"
              :permissions="[
                'bb.policies.update'
              ]"
            >
              <NInputNumber
                :value="maximumResultRows"
                :disabled="!hasQueryPolicyFeature || slotProps.disabled"
                class="w-60"
                :min="0"
                :precision="0"
                @update:value="handleMaxRowsInput"
              >
                <template #suffix>{{
                  $t("settings.general.workspace.maximum-sql-result.rows.rows")
                }}</template>
              </NInputNumber>
            </PermissionGuardWrapper>
          </div>
        </div>
      </div>
    </ComponentPermissionGuard>

    <div>
      <div class="flex items-center gap-x-2">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :project="project"
          :permissions="[
            'bb.projects.update'
          ]"
        >
          <Switch
            v-model:value="allowRequestRole"
            :text="true"
            :disabled="
              slotProps.disabled ||
              loading
            "
          />
        </PermissionGuardWrapper>
        <div class="textlabel flex items-center gap-x-2">
          {{
            $t(
              "project.settings.issue-related.allow-request-role.self"
            )
          }}
        </div>
        <ApprovalFlowIndicator
          :source="WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE"
        />
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t(
            "project.settings.issue-related.allow-request-role.description"
          )
        }}
      </div>
    </div>

    <div>
      <div class="flex items-center gap-x-2">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :project="project"
          :permissions="[
            'bb.projects.update'
          ]"
        >
          <Switch
            v-model:value="allowJustInTimeAccess"
            :text="true"
            :disabled="
              slotProps.disabled ||
              loading
            "
          />
        </PermissionGuardWrapper>
        <div class="textlabel flex items-center gap-x-2">
          {{
            $t(
              "project.settings.issue-related.allow-jit.self"
            )
          }}
        </div>
        <ApprovalFlowIndicator
          :source="WorkspaceApprovalSetting_Rule_Source.REQUEST_ACCESS"
        />
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{
          $t(
            "project.settings.issue-related.allow-jit.description"
          )
        }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { NInputNumber } from "naive-ui";
import { computed, ref, watch } from "vue";
import { FeatureBadge } from "@/components/FeatureGuard";
import ComponentPermissionGuard from "@/components/Permission/ComponentPermissionGuard.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { SQLReviewForResource } from "@/components/SQLReview";
import { Switch } from "@/components/v2";
import {
  featureToRef,
  usePolicyByParentAndType,
  usePolicyV1Store,
  useProjectV1Store,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import ApprovalFlowIndicator from "./ApprovalFlowIndicator.vue";

const props = defineProps<{
  project: Project;
}>();

const projectStore = useProjectV1Store();
const policyV1Store = usePolicyV1Store();
const hasQueryPolicyFeature = featureToRef(PlanFeature.FEATURE_QUERY_POLICY);

const allowRequestRole = ref<boolean>(props.project.allowRequestRole);
const allowJustInTimeAccess = ref<boolean>(props.project.allowJustInTimeAccess);
const loading = ref<boolean>(false);

const sqlReviewForResourceRef =
  ref<InstanceType<typeof SQLReviewForResource>>();

const { ready } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.project.name,
    policyType: PolicyType.DATA_QUERY,
  }))
);

const policyPayload = computed(() => {
  return policyV1Store.getQueryDataPolicyByParent(props.project.name);
});

// Maximum result rows (project-level)
const getInitialMaxRows = () => {
  const rows = Number(policyPayload.value.maximumResultRows);
  return rows < 0 ? 0 : rows;
};
const maximumResultRows = ref<number>(getInitialMaxRows());

watch(
  () => ready.value,
  (ready) => {
    if (ready) {
      maximumResultRows.value = getInitialMaxRows();
    }
  }
);

const handleMaxRowsInput = (value: number | null) => {
  if (value === null || value === undefined) return;
  maximumResultRows.value = value;
};

const maxRowsDirty = computed(
  () => maximumResultRows.value !== getInitialMaxRows()
);

const isDirty = computed(
  () =>
    sqlReviewForResourceRef.value?.isDirty ||
    maxRowsDirty.value ||
    allowRequestRole.value !== props.project.allowRequestRole ||
    allowJustInTimeAccess.value !== props.project.allowJustInTimeAccess
);

const updateMask = computed(() => {
  const mask: string[] = [];
  if (allowRequestRole.value !== props.project.allowRequestRole) {
    mask.push("allow_request_role");
  }
  if (allowJustInTimeAccess.value !== props.project.allowJustInTimeAccess) {
    mask.push("allow_just_in_time_access");
  }
  return mask;
});

const doUpdateProject = async () => {
  if (loading.value) {
    return;
  }
  if (updateMask.value.length === 0) {
    return;
  }
  loading.value = true;
  const projectPatch = {
    ...cloneDeep(props.project),
    allowRequestRole: allowRequestRole.value,
    allowJustInTimeAccess: allowJustInTimeAccess.value,
  };
  try {
    await projectStore.updateProject(projectPatch, updateMask.value);
  } finally {
    loading.value = false;
  }
};

const updateMaxRows = async () => {
  await policyV1Store.upsertPolicy({
    parentPath: props.project.name,
    policy: {
      type: PolicyType.DATA_QUERY,
      resourceType: PolicyResourceType.PROJECT,
      policy: {
        case: "queryDataPolicy",
        value: create(QueryDataPolicySchema, {
          ...policyPayload.value,
          maximumResultRows: maximumResultRows.value,
        }),
      },
    },
  });
};

const onUpdate = async () => {
  if (sqlReviewForResourceRef.value?.isDirty) {
    await sqlReviewForResourceRef.value.update();
  }
  if (maxRowsDirty.value) {
    await updateMaxRows();
  }
  await doUpdateProject();
};

const resetState = () => {
  sqlReviewForResourceRef.value?.revert();
  maximumResultRows.value = getInitialMaxRows();
  allowRequestRole.value = props.project.allowRequestRole;
  allowJustInTimeAccess.value = props.project.allowJustInTimeAccess;
};

defineExpose({
  isDirty,
  update: onUpdate,
  revert: resetState,
});
</script>
