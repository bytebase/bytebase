<template>
  <div>
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">{{
        $t("settings.general.workspace.maximum-role-expiration.self")
      }}</span>
    </p>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.maximum-role-expiration.description") }}
    </p>
    <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
      <NInputNumber
        v-model:value="state.inputValue"
        class="w-60"
        :disabled="!allowEdit || state.neverExpire"
        :min="1"
        :precision="0"
      >
        <template #suffix>
          {{ $t("settings.general.workspace.maximum-role-expiration.days") }}
        </template>
      </NInputNumber>
      <NCheckbox
        :disabled="!allowEdit"
        v-model:checked="state.neverExpire"
        style="margin-right: 12px"
      >
        {{
          $t("settings.general.workspace.maximum-role-expiration.never-expires")
        }}
      </NCheckbox>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { DurationSchema, FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { NCheckbox, NInputNumber } from "naive-ui";
import { computed, reactive } from "vue";
import { useSettingV1Store } from "@/store/modules/v1/setting";

const DEFAULT_EXPIRATION_DAYS = 90;

interface LocalState {
  inputValue: number;
  neverExpire: boolean;
}

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    inputValue: DEFAULT_EXPIRATION_DAYS,
    neverExpire: true,
  };
  const seconds = settingV1Store.workspaceProfileSetting?.maximumRoleExpiration
    ?.seconds
    ? Number(
        settingV1Store.workspaceProfileSetting.maximumRoleExpiration.seconds
      )
    : undefined;
  if (seconds && seconds > 0) {
    defaultState.inputValue =
      Math.floor(seconds / (60 * 60 * 24)) || DEFAULT_EXPIRATION_DAYS;
    defaultState.neverExpire = false;
  }
  return defaultState;
};

defineProps<{
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(getInitialState());

const handleSettingChange = async () => {
  let seconds = -1;
  if (!state.neverExpire) {
    seconds = state.inputValue * 24 * 60 * 60;
  }
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      maximumRoleExpiration: create(DurationSchema, {
        seconds: BigInt(seconds),
        nanos: 0,
      }),
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.maximum_role_expiration"],
    }),
  });
  Object.assign(state, getInitialState());
};

defineExpose({
  isDirty: computed(() => !isEqual(state, getInitialState())),
  revert: () => {
    Object.assign(state, getInitialState());
  },
  update: handleSettingChange,
});
</script>
