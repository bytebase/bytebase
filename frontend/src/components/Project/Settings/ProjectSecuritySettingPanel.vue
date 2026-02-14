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

        <MaximumSQLResultSizeSetting
          ref="maximumSQLResultSizeSettingRef"
          :resource="project.name"
          :policy="policyPayload"
        />
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

    <div v-if="isDev()">
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
import { cloneDeep } from "lodash-es";
import { computed, ref, watch } from "vue";
import MaximumSQLResultSizeSetting from "@/components/GeneralSetting/MaximumSQLResultSizeSetting.vue";
import ComponentPermissionGuard from "@/components/Permission/ComponentPermissionGuard.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { SQLReviewForResource } from "@/components/SQLReview";
import { Switch } from "@/components/v2";
import {
  usePolicyByParentAndType,
  usePolicyV1Store,
  useProjectV1Store,
} from "@/store";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { isDev } from "@/utils";
import ApprovalFlowIndicator from "./ApprovalFlowIndicator.vue";

const props = defineProps<{
  project: Project;
}>();

const projectStore = useProjectV1Store();
const policyV1Store = usePolicyV1Store();

const allowRequestRole = ref<boolean>(props.project.allowRequestRole);
const allowJustInTimeAccess = ref<boolean>(props.project.allowJustInTimeAccess);
const loading = ref<boolean>(false);

const sqlReviewForResourceRef =
  ref<InstanceType<typeof SQLReviewForResource>>();
const maximumSQLResultSizeSettingRef =
  ref<InstanceType<typeof MaximumSQLResultSizeSetting>>();

const { ready } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.project.name,
    policyType: PolicyType.DATA_QUERY,
  }))
);

watch(
  () => ready.value,
  (ready) => {
    if (ready) {
      maximumSQLResultSizeSettingRef.value?.revert();
    }
  }
);

const policyPayload = computed(() => {
  return policyV1Store.getQueryDataPolicyByParent(props.project.name);
});

const isDirty = computed(
  () =>
    sqlReviewForResourceRef.value?.isDirty ||
    maximumSQLResultSizeSettingRef.value?.isDirty ||
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

const onUpdate = async () => {
  if (sqlReviewForResourceRef.value?.isDirty) {
    await sqlReviewForResourceRef.value.update();
  }
  if (maximumSQLResultSizeSettingRef.value?.isDirty) {
    await maximumSQLResultSizeSettingRef.value.update();
  }
  await doUpdateProject();
};

const resetState = () => {
  sqlReviewForResourceRef.value?.revert();
  maximumSQLResultSizeSettingRef.value?.revert();
  allowRequestRole.value = props.project.allowRequestRole;
  allowJustInTimeAccess.value = props.project.allowJustInTimeAccess;
};

defineExpose({
  isDirty,
  update: onUpdate,
  revert: resetState,
});
</script>
