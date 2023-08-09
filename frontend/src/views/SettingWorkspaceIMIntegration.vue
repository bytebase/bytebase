<template>
  <div class="w-full mt-4 space-y-4">
    <FeatureAttentionForInstanceLicense feature="bb.feature.im.approval" />
    <div class="textinfolabel">
      {{ $t("settings.im-integration.description") }}
      <a
        class="normal-link inline-flex items-center"
        href="https://www.bytebase.com/docs/administration/external-approval?source=console"
        target="__BLANK"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4 ml-1" />
      </a>
    </div>
    <div class="w-full flex flex-col justify-start items-start space-y-2">
      <div class="w-full flex flex-row justify-start items-center">
        <div class="flex flex-row justify-start items-center">
          <img class="w-10 h-auto" src="../assets/feishu-logo.webp" alt="" />
          <span class="ml-2 text-lg font-medium">{{
            $t("common.feishu")
          }}</span>
          <FeatureBadge
            :feature="'bb.feature.im.approval'"
            custom-class="ml-2"
          />
        </div>
        <button
          v-if="!state.feishuSetting"
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click.prevent="createFeishuIntegration"
        >
          {{ $t("common.create") }}
        </button>
      </div>
      <div
        v-if="state.feishuSetting"
        class="w-full flex flex-col justify-start items-start space-y-2"
      >
        <div class="textlabel">{{ $t("common.application") }} ID</div>
        <BBTextField
          class="w-128 max-w-full mb-2"
          :placeholder="'ex. cli_a3c48b4c45f933xz'"
          :value="state.feishuSetting.appId"
          @input="(e: any) => state.feishuSetting!.appId = e.target.value"
        />
        <div class="mt-4 textlabel">Secret</div>
        <BBTextField
          class="w-128 max-w-full mb-2"
          :placeholder="'ex. MTOc5YmoRYJyDfRXHUzSBeXzTu3w3I3G'"
          :value="state.feishuSetting.appSecret"
          @input="(e: any) => state.feishuSetting!.appSecret = e.target.value"
        />
        <div
          class="!mt-4 !mb-2 w-128 max-w-full flex flex-row justify-start items-center"
        >
          <span class="textlabel mr-4">{{
            $t("settings.im-integration.enable")
          }}</span>
          <BBSwitch
            v-if="state.feishuSetting.externalApproval"
            :value="state.feishuSetting.externalApproval.enabled"
            @toggle="onFeishuIntegrationEnableToggle"
          />
        </div>
        <div class="flex flex-row justify-center">
          <button
            type="button"
            class="btn-primary inline-flex justify-center py-2 px-4"
            :disabled="!allowFeishuActionButton || state.isLoading"
            @click.prevent="updateFeishuIntegration"
          >
            {{ feishuActionButtonText }}
          </button>
          <BBSpin v-if="state.isLoading" class="ml-1" />
        </div>
      </div>
    </div>
  </div>

  <FeatureModal
    feature="bb.feature.im.approval"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBSwitch } from "@/bbkit";
import { featureToRef, pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  AppIMSetting,
  AppIMSetting_IMType,
} from "@/types/proto/v1/setting_service";

interface LocalState {
  originFeishuSetting?: AppIMSetting;
  feishuSetting?: AppIMSetting;
  showFeatureModal: boolean;
  isLoading: boolean;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  isLoading: false,
});
const settingV1Store = useSettingV1Store();
const hasIMApprovalFeature = featureToRef("bb.feature.im.approval");

const feishuActionButtonText = computed(() => {
  return state.originFeishuSetting === undefined
    ? t("common.create")
    : t("common.update");
});

const allowFeishuActionButton = computed(() => {
  return !isEqual(state.originFeishuSetting, state.feishuSetting);
});

onMounted(() => {
  const setting = settingV1Store.getSettingByName("bb.app.im");
  const appFeishuValue = setting?.value?.appImSettingValue;
  if (appFeishuValue?.imType === AppIMSetting_IMType.FEISHU) {
    state.originFeishuSetting = cloneDeep(appFeishuValue);
    state.feishuSetting = appFeishuValue;
  }
});

const onFeishuIntegrationEnableToggle = (status: boolean) => {
  if (state.feishuSetting && state.feishuSetting.externalApproval) {
    state.feishuSetting.externalApproval.enabled = status;
  }
};

const createFeishuIntegration = () => {
  if (!hasIMApprovalFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  state.feishuSetting = {
    imType: AppIMSetting_IMType.FEISHU,
    appId: "",
    appSecret: "",
    externalApproval: {
      enabled: true,
      approvalDefinitionId: "",
    },
  };
};

const updateFeishuIntegration = async () => {
  if (!hasIMApprovalFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  state.isLoading = true;
  try {
    await settingV1Store.upsertSetting({
      name: "bb.app.im",
      value: {
        appImSettingValue: state.feishuSetting,
      },
    });
  } catch (error) {
    state.isLoading = false;
    return;
  }

  state.isLoading = false;
  state.originFeishuSetting = cloneDeep(state.feishuSetting);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.im-integration.feishu-updated-tip"),
  });
};
</script>
