<template>
  <div id="audit-log-stdout" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center gap-x-2">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.audit-log-stdout.self") }}
        </h1>
        <FeatureBadge :feature="PlanFeature.FEATURE_AUDIT_LOG" />
      </div>
    </div>
    <div class="flex-1 lg:px-4">
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.enableAuditLogStdout"
          :text="true"
          :disabled="!allowEdit || !hasAuditLogFeature"
        />
        <span class="text-sm">
          {{ $t("settings.general.workspace.audit-log-stdout.enable") }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{ $t("settings.general.workspace.audit-log-stdout.description") }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { computed, reactive } from "vue";
import { Switch } from "@/components/v2";
import { featureToRef, useSettingV1Store } from "@/store";
import type { WorkspaceProfileSetting } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";

interface LocalState {
  enableAuditLogStdout: boolean;
}

defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const hasAuditLogFeature = featureToRef(PlanFeature.FEATURE_AUDIT_LOG);

const getInitialState = (): LocalState => {
  return {
    enableAuditLogStdout:
      settingV1Store.workspaceProfileSetting?.enableAuditLogStdout ?? false,
  };
};

const state = reactive(getInitialState());
const originalState = reactive(getInitialState());

const isDirty = computed(() => {
  return !isEqual(state, originalState);
});

const revert = () => {
  Object.assign(state, originalState);
};

const update = async () => {
  const payload: Partial<WorkspaceProfileSetting> = {
    enableAuditLogStdout: state.enableAuditLogStdout,
  };

  await settingV1Store.updateWorkspaceProfile({
    payload,
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.enable_audit_log_stdout"],
    }),
  });

  Object.assign(originalState, state);
};

defineExpose({
  isDirty,
  revert,
  update,
  title: "Audit Log Stdout",
});
</script>
