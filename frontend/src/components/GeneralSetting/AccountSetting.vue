<template>
  <div id="account" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.account") }}
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
            :value="disallowSignupEnabled"
            :text="true"
            :disabled="!allowEdit || !hasDisallowSignupFeature"
            @update:value="handleDisallowSignupToggle"
          />
          <span class="textlabel">
            {{ $t("settings.general.workspace.disallow-signup.enable") }}
          </span>
        </div>
        <div class="mt-1 mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.disallow-signup.description") }}
        </div>
      </div>

      <NDivider v-if="!isSaaSMode" />

      <PasswordRestrictionSetting :allow-edit="allowEdit" />
      <NDivider />

      <div>
        <div class="mb-7 mt-4 lg:mt-0">
          <div class="flex items-center gap-x-2">
            <Switch
              :value="require2FAEnabled"
              :text="true"
              :disabled="!allowEdit || !has2FAFeature"
              @update:value="handleRequire2FAToggle"
            />
            <div class="textlabel flex items-center space-x-2">
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
              :value="disallowPasswordSignin"
              :text="true"
              :disabled="!allowEdit || !hasDisallowPasswordSigninFeature"
              @update:value="handleDisallowPasswordSigninToggle"
            />
            <div class="textlabel flex items-center space-x-2">
              {{
                $t("settings.general.workspace.disallow-password-signin.enable")
              }}
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

      <SignInFrequencySetting :allow-edit="allowEdit" />
    </div>
  </div>

  <FeatureModal
    :open="!!state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { NDivider } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { Switch } from "@/components/v2";
import {
  featureToRef,
  pushNotification,
  useActuatorV1Store,
  useIdentityProviderStore,
} from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import type { FeatureType } from "@/types";
import { State } from "@/types/proto/v1/common";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";
import PasswordRestrictionSetting from "./PasswordRestrictionSetting.vue";
import SignInFrequencySetting from "./SignInFrequencySetting.vue";

interface LocalState {
  featureNameForModal?: FeatureType;
}

defineProps<{
  allowEdit: boolean;
}>();

const state = reactive<LocalState>({});
const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const actuatorStore = useActuatorV1Store();

const { isSaaSMode } = storeToRefs(actuatorStore);
const has2FAFeature = featureToRef("bb.feature.2fa");
const hasDisallowSignupFeature = featureToRef("bb.feature.disallow-signup");
const hasDisallowPasswordSigninFeature = featureToRef(
  "bb.feature.disallow-password-signin"
);

const disallowSignupEnabled = computed((): boolean => {
  return settingV1Store.workspaceProfileSetting?.disallowSignup ?? false;
});
const require2FAEnabled = computed((): boolean => {
  return settingV1Store.workspaceProfileSetting?.require2fa ?? false;
});
const disallowPasswordSignin = computed((): boolean => {
  return (
    settingV1Store.workspaceProfileSetting?.disallowPasswordSignin ?? false
  );
});

const handleDisallowSignupToggle = async (on: boolean) => {
  if (!hasDisallowSignupFeature.value && on) {
    state.featureNameForModal = "bb.feature.disallow-signup";
    return;
  }
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      disallowSignup: on,
    },
    updateMask: ["value.workspace_profile_setting_value.disallow_signup"],
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};

const handleRequire2FAToggle = async (on: boolean) => {
  if (!has2FAFeature.value && on) {
    state.featureNameForModal = "bb.feature.2fa";
    return;
  }

  await settingV1Store.updateWorkspaceProfile({
    payload: {
      require2fa: on,
    },
    updateMask: ["value.workspace_profile_setting_value.require_2fa"],
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};

const handleDisallowPasswordSigninToggle = async (on: boolean) => {
  if (!hasDisallowPasswordSigninFeature.value && on) {
    state.featureNameForModal = "bb.feature.disallow-password-signin";
    return;
  }

  if (on) {
    const idpStore = useIdentityProviderStore();
    const idpList = await idpStore.fetchIdentityProviderList();
    if (idpList.filter((idp) => idp.state === State.ACTIVE).length === 0) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t(
          "settings.general.workspace.disallow-password-signin.require-sso-setup"
        ),
      });
      return;
    }
  }

  await settingV1Store.updateWorkspaceProfile({
    payload: {
      disallowPasswordSignin: on,
    },
    updateMask: [
      "value.workspace_profile_setting_value.disallow_password_signin",
    ],
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};
</script>
