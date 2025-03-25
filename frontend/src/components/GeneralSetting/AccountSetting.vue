<template>
  <div id="account" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>

    <div class="flex-1 lg:px-4">
      <div v-if="!isSaaSMode" class="mb-7 mt-4 lg:mt-0">
        <div class="flex items-center gap-x-2">
          <Switch
            v-model:value="state.disallowSignup"
            :text="true"
            :disabled="!allowEdit || !hasDisallowSignupFeature"
          />
          <div class="font-medium flex items-center gap-x-2">
            {{ $t("settings.general.workspace.disallow-signup.enable") }}
            <FeatureBadge feature="bb.feature.disallow-signup" />
          </div>
        </div>
        <div class="mt-1 mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.disallow-signup.description") }}
        </div>
      </div>

      <NDivider v-if="!isSaaSMode" />

      <PasswordRestrictionSetting
        ref="passwordRestrictionSettingRef"
        :allow-edit="allowEdit"
      />
      <NDivider />

      <div>
        <div class="mb-7 mt-4 lg:mt-0">
          <div class="flex items-center gap-x-2">
            <Switch
              v-model:value="state.require2fa"
              :text="true"
              :disabled="!allowEdit || !has2FAFeature"
            />
            <div class="font-medium flex items-center gap-x-2">
              {{ $t("settings.general.workspace.require-2fa.enable") }}
              <FeatureBadge feature="bb.feature.2fa" />
            </div>
          </div>
          <div class="mt-1 mb-3 text-sm text-gray-400">
            {{ $t("settings.general.workspace.require-2fa.description") }}
          </div>
        </div>
        <div class="mb-7 mt-4 lg:mt-0">
          <div class="flex items-center gap-x-2">
            <Switch
              v-model:value="state.disallowPasswordSignin"
              :text="true"
              :disabled="
                !allowEdit ||
                !hasDisallowPasswordSigninFeature ||
                (!state.disallowPasswordSignin && !existActiveIdentityProvider)
              "
            />
            <div class="font-medium flex items-center gap-x-2">
              {{
                $t("settings.general.workspace.disallow-password-signin.enable")
              }}
              <NTooltip
                v-if="
                  allowEdit &&
                  hasDisallowPasswordSigninFeature &&
                  !state.disallowPasswordSignin &&
                  !existActiveIdentityProvider
                "
              >
                <template #trigger>
                  <TriangleAlertIcon class="w-4 text-warning" />
                </template>
                {{
                  $t(
                    "settings.general.workspace.disallow-password-signin.require-sso-setup"
                  )
                }}
              </NTooltip>
              <FeatureBadge feature="bb.feature.disallow-password-signin" />
            </div>
          </div>
          <div class="mt-1 mb-3 text-sm text-gray-400">
            {{
              $t(
                "settings.general.workspace.disallow-password-signin.description"
              )
            }}
          </div>
        </div>
      </div>
      <NDivider />

      <SignInFrequencySetting
        ref="signInFrequencySettingRef"
        :allow-edit="allowEdit"
      />
    </div>
  </div>

  <FeatureModal
    :open="!!state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { TriangleAlertIcon } from "lucide-vue-next";
import { NDivider, NTooltip } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive, ref, watchEffect } from "vue";
import { Switch } from "@/components/v2";
import {
  featureToRef,
  useActuatorV1Store,
  useIdentityProviderStore,
} from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import type { FeatureType } from "@/types";
import { State } from "@/types/proto/v1/common";
import { type WorkspaceProfileSetting } from "@/types/proto/v1/setting_service";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";
import PasswordRestrictionSetting from "./PasswordRestrictionSetting.vue";
import SignInFrequencySetting from "./SignInFrequencySetting.vue";

interface LocalState {
  featureNameForModal?: FeatureType;
  disallowSignup: boolean;
  require2fa: boolean;
  disallowPasswordSignin: boolean;
}

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const actuatorStore = useActuatorV1Store();
const idpStore = useIdentityProviderStore();
const passwordRestrictionSettingRef =
  ref<InstanceType<typeof PasswordRestrictionSetting>>();
const signInFrequencySettingRef =
  ref<InstanceType<typeof SignInFrequencySetting>>();

const getInitialState = (): LocalState => {
  return {
    disallowSignup:
      settingV1Store.workspaceProfileSetting?.disallowSignup ?? false,
    require2fa: settingV1Store.workspaceProfileSetting?.require2fa ?? false,
    disallowPasswordSignin:
      settingV1Store.workspaceProfileSetting?.disallowPasswordSignin ?? false,
  };
};

const state = reactive<LocalState>({
  ...getInitialState(),
});

const { isSaaSMode } = storeToRefs(actuatorStore);
const has2FAFeature = featureToRef("bb.feature.2fa");
const hasDisallowSignupFeature = featureToRef("bb.feature.disallow-signup");
const hasDisallowPasswordSigninFeature = featureToRef(
  "bb.feature.disallow-password-signin"
);

watchEffect(async () => {
  await idpStore.fetchIdentityProviderList();
});

const existActiveIdentityProvider = computed(() => {
  return (
    idpStore.identityProviderList.filter((idp) => idp.state === State.ACTIVE)
      .length > 0
  );
});

const isDirty = computed(() => {
  return (
    passwordRestrictionSettingRef.value?.isDirty ||
    signInFrequencySettingRef.value?.isDirty ||
    !isEqual(state, getInitialState())
  );
});

const onUpdate = async () => {
  if (passwordRestrictionSettingRef.value?.isDirty) {
    await passwordRestrictionSettingRef.value.update();
  }
  if (signInFrequencySettingRef.value?.isDirty) {
    await signInFrequencySettingRef.value.update();
  }

  const updateMasks = [];
  const payload: Partial<WorkspaceProfileSetting> = {
    disallowSignup: state.disallowSignup,
    require2fa: state.require2fa,
    disallowPasswordSignin: state.disallowPasswordSignin,
  };
  if (
    state.disallowSignup !==
    settingV1Store.workspaceProfileSetting?.disallowSignup
  ) {
    updateMasks.push("value.workspace_profile_setting_value.disallow_signup");
  }
  if (state.require2fa !== settingV1Store.workspaceProfileSetting?.require2fa) {
    updateMasks.push("value.workspace_profile_setting_value.require_2fa");
  }
  if (
    state.disallowPasswordSignin !==
    settingV1Store.workspaceProfileSetting?.disallowPasswordSignin
  ) {
    updateMasks.push(
      "value.workspace_profile_setting_value.disallow_password_signin"
    );
  }

  if (updateMasks.length > 0) {
    await settingV1Store.updateWorkspaceProfile({
      payload,
      updateMask: updateMasks,
    });
  }
};

defineExpose({
  isDirty,
  title: props.title,
  update: onUpdate,
  revert: () => {
    Object.assign(state, getInitialState());
    passwordRestrictionSettingRef.value?.revert();
    signInFrequencySettingRef.value?.revert();
  },
});
</script>
