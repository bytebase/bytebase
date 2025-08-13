<template>
  <div id="network" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <h1 class="text-2xl font-bold">
        {{ title }}
      </h1>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>
    <div class="flex-1 lg:px-4">
      <div class="mt-4 lg:mt-0">
        <label class="flex items-center gap-x-2">
          <span class="font-medium">{{
            $t("settings.general.workspace.external-url.self")
          }}</span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.external-url.description") }}
          <LearnMoreLink
            url="https://docs.bytebase.com/get-started/self-host/external-url?source=console"
          />
        </div>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <NInput
              v-model:value="state.externalUrl"
              class="mb-4 w-full"
              :disabled="!allowEdit || isSaaSMode"
            />
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{ $t("settings.general.workspace.only-admin-can-edit") }}
          </span>
        </NTooltip>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { NInput, NTooltip } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import { useSettingV1Store, useActuatorV1Store } from "@/store";
import LearnMoreLink from "../LearnMoreLink.vue";

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

interface LocalState {
  externalUrl: string;
}

const settingV1Store = useSettingV1Store();
const actuatorV1Store = useActuatorV1Store();

const getInitialState = (): LocalState => {
  return {
    externalUrl: settingV1Store.workspaceProfileSetting?.externalUrl ?? "",
  };
};

const state = reactive<LocalState>(getInitialState());

const { isSaaSMode } = storeToRefs(actuatorV1Store);

const allowSave = computed((): boolean => {
  const externalUrlChanged =
    state.externalUrl !==
    (settingV1Store.workspaceProfileSetting?.externalUrl ?? "");
  return externalUrlChanged;
});

const updateNetworkSetting = async () => {
  if (!allowSave.value) {
    return;
  }

  await settingV1Store.updateWorkspaceProfile({
    payload: {
      externalUrl: state.externalUrl,
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile_setting_value.external_url"],
    }),
  });

  state.externalUrl = settingV1Store.workspaceProfileSetting?.externalUrl ?? "";
};

defineExpose({
  title: props.title,
  isDirty: allowSave,
  update: updateNetworkSetting,
  revert: () => {
    return Object.assign(state, getInitialState());
  },
});
</script>
