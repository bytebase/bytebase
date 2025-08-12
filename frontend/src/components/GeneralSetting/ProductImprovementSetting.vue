<template>
  <div id="product-improvement" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <h1 class="text-2xl font-bold">
        {{ $t("settings.general.workspace.product-improvement.self") }}
      </h1>
    </div>
    <div class="flex-1 lg:px-4">
      <div class="flex items-center gap-x-2">
        <Switch
          v-model:value="state.enableMetricCollection"
          :text="true"
          :disabled="!allowEdit"
        />
        <span class="text-sm">
          {{ $t("settings.general.workspace.product-improvement.participate") }}
        </span>
      </div>
      <div class="mt-1 text-sm text-gray-400">
        {{ $t("settings.general.workspace.product-improvement.description") }}
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
import { useSettingV1Store } from "@/store";
import type { WorkspaceProfileSetting } from "@/types/proto-es/v1/setting_service_pb";

interface LocalState {
  enableMetricCollection: boolean;
}

defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();

const getInitialState = (): LocalState => {
  return {
    enableMetricCollection:
      settingV1Store.workspaceProfileSetting?.enableMetricCollection ?? true,
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
    enableMetricCollection: state.enableMetricCollection,
  };

  await settingV1Store.updateWorkspaceProfile({
    payload,
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile_setting_value.enable_metric_collection"],
    }),
  });

  Object.assign(originalState, state);
};

defineExpose({
  isDirty,
  revert,
  update,
  title: "Product Improvement",
});
</script>
