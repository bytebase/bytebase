<template>
  <div>
    <h3
      id="domain-restriction"
      class="font-medium flex flex-row justify-start items-center"
    >
      <span class="mr-2">{{
        $t("settings.general.workspace.domain-restriction.self")
      }}</span>
    </h3>
    <p class="text-sm text-gray-400 mt-1">
      {{ $t("settings.general.workspace.domain-restriction.description") }}
    </p>
    <div class="w-full flex flex-col gap-2 mt-2">
      <NInput
        v-model:value="state.domain"
        :readonly="!allowEdit"
        :placeholder="
          $t(
            'settings.general.workspace.domain-restriction.domain-input-placeholder'
          )
        "
        type="text"
      />
      <div class="w-full flex flex-row justify-between items-center">
        <NCheckbox
          v-model:checked="state.enableRestriction"
          :disabled="!state.domain"
          :readonly="!allowEdit"
        >
          <p class="font-medium">
            {{
              $t(
                "settings.general.workspace.domain-restriction.members-restriction.self"
              )
            }}
          </p>
          <p class="text-sm text-gray-400 leading-tight">
            {{
              $t(
                "settings.general.workspace.domain-restriction.members-restriction.description"
              )
            }}
          </p>
        </NCheckbox>
      </div>
    </div>
    <div class="flex justify-end mt-2">
      <NButton
        type="primary"
        :disabled="!allowSaveUpdates"
        @click="saveDomainRestrictionSettings"
      >
        {{ $t("common.update") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head, isEqual } from "lodash-es";
import { NButton, NCheckbox, NInput } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    domain: "",
    enableRestriction: false,
  };
  if (
    Array.isArray(settingV1Store.workspaceProfileSetting?.domains) &&
    settingV1Store.workspaceProfileSetting?.domains.length > 0
  ) {
    defaultState.domain =
      head(settingV1Store.workspaceProfileSetting?.domains) || "";
    defaultState.enableRestriction =
      settingV1Store.workspaceProfileSetting?.enforceIdentityDomain || false;
  }
  return defaultState;
};

interface LocalState {
  domain: string;
  enableRestriction: boolean;
}

defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const state = reactive<LocalState>(getInitialState());

const allowSaveUpdates = computed(() => {
  return !isEqual(state, getInitialState());
});

const saveDomainRestrictionSettings = async () => {
  if (state.domain.length === 0) {
    state.enableRestriction = false;
  }
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      domains: state.domain ? [state.domain] : [],
      enforceIdentityDomain: state.enableRestriction,
    },
    updateMask: [
      "value.workspace_profile_setting_value.domains",
      "value.workspace_profile_setting_value.enforce_identity_domain",
    ],
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};
</script>
