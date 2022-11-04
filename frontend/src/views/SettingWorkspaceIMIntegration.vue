<template>
  <div class="w-full mt-4 space-y-4">
    <div class="w-full flex flex-col justify-start items-start space-y-2">
      <div class="w-full flex flex-row justify-start items-center">
        <div class="flex flex-row justify-start items-center">
          <img class="w-10 h-auto" src="../assets/feishu-logo.png" alt="" />
          <span class="ml-2 text-lg font-medium">{{
            $t("common.feishu")
          }}</span>
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
        <button
          type="button"
          class="btn-primary inline-flex justify-center py-2 px-4 mt-2"
          @click.prevent="updateFeishuIntegration"
        >
          {{ $t("common.update") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive } from "vue";
import { pushNotification, useSettingStore } from "@/store";
import { SettingAppIMValue } from "@/types/setting";
import { useI18n } from "vue-i18n";

interface LocalState {
  feishuSetting?: SettingAppIMValue;
}

const { t } = useI18n();
const state = reactive<LocalState>({});
const settingStore = useSettingStore();

onMounted(() => {
  const setting = settingStore.getSettingByName("bb.app.im");
  if (setting) {
    const appIMValue = JSON.parse(setting.value || "{}") as SettingAppIMValue;
    if (appIMValue.imType === "im.feishu") {
      state.feishuSetting = appIMValue;
    }
  }
});

const createFeishuIntegration = async () => {
  state.feishuSetting = {
    imType: "im.feishu",
    appId: "",
    appSecret: "",
    externalApproval: {
      enabled: true,
    },
  };
  await settingStore.updateSettingByName({
    name: "bb.app.im",
    value: JSON.stringify(state.feishuSetting),
  });
};

const updateFeishuIntegration = async () => {
  await settingStore.updateSettingByName({
    name: "bb.app.im",
    value: JSON.stringify(state.feishuSetting),
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.im-integration.updated-tip"),
  });
};
</script>
