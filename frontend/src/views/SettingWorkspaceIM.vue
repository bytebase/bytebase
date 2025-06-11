<template>
  <div class="w-full space-y-4">
    <div class="textinfolabel">
      {{ $t("settings.im-integration.description") }}
      <a
        class="normal-link inline-flex items-center"
        href="https://docs.bytebase.com/change-database/webhook?source=console"
        target="__BLANK"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4 ml-1" />
      </a>
    </div>
    <NTabs v-model:value="state.selectedTab" type="line" animated>
      <NTabPane v-for="item in imList" :key="item.type" :name="item.type">
        <template #tab>
          <div class="flex items-center gap-x-2">
            <WebhookTypeIcon :type="item.type" class="h-5 w-5" />
            {{ item.name }}
          </div>
        </template>
        <div>
          <BBAttention v-if="item.enabled" class="mt-2 mb-4" type="success">
            <template #default>IM App Enabled</template>
          </BBAttention>
          <component :is="() => item.render()" />
        </div>
      </NTabPane>
    </NTabs>
    <div class="flex items-center justify-end space-x-2 pt-2">
      <NButton
        v-if="dataChanged"
        :disabled="state.loading"
        @click="discardChanges"
      >
        {{ $t("common.discard-changes") }}
      </NButton>
      <NButton
        type="primary"
        :disabled="!allowEdit || !canSave"
        :loading="state.loading"
        @click="onSave"
      >
        {{ $t("common.save") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { NTabs, NTabPane, NButton } from "naive-ui";
import { computed, watch, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention } from "@/bbkit";
import BBTextField from "@/bbkit/BBTextField.vue";
import WebhookTypeIcon from "@/components/Project/WebhookTypeIcon.vue";
import { useSettingV1Store, pushNotification } from "@/store";
import { Webhook_Type } from "@/types/proto/v1/project_service";
import {
  AppIMSetting,
  AppIMSetting_Feishu,
  AppIMSetting_Slack,
  AppIMSetting_Lark,
  AppIMSetting_Wecom,
  AppIMSetting_DingTalk,
  Setting_SettingName,
} from "@/types/proto/v1/setting_service";

interface LocalState {
  selectedTab: Webhook_Type;
  loading: boolean;
  setting: AppIMSetting;
}

defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedTab: Webhook_Type.SLACK,
  loading: false,
  setting: AppIMSetting.fromPartial({}),
});

const settingStore = useSettingV1Store();

const imSetting = computed(
  () =>
    settingStore.getSettingByName(Setting_SettingName.APP_IM)?.value
      ?.appImSettingValue ?? AppIMSetting.fromPartial({})
);

watch(
  () => imSetting.value,
  (setting) => {
    state.setting = cloneDeep(setting);
  },
  { once: true, immediate: true }
);

watch(
  () => state.selectedTab,
  (tab) => {
    switch (tab) {
      case Webhook_Type.SLACK:
        if (!state.setting.slack) {
          state.setting.slack = AppIMSetting_Slack.fromPartial({});
        }
        break;
      case Webhook_Type.FEISHU:
        if (!state.setting.feishu) {
          state.setting.feishu = AppIMSetting_Feishu.fromPartial({});
        }
        break;
      case Webhook_Type.WECOM:
        if (!state.setting.wecom) {
          state.setting.wecom = AppIMSetting_Wecom.fromPartial({});
        }
        break;
      case Webhook_Type.LARK:
        if (!state.setting.lark) {
          state.setting.lark = AppIMSetting_Lark.fromPartial({});
        }
        break;
      case Webhook_Type.DINGTALK:
        if (!state.setting.dingtalk) {
          state.setting.dingtalk = AppIMSetting_DingTalk.fromPartial({});
        }
        break;
    }
  },
  { immediate: true }
);

const imList = computed(() => {
  return [
    {
      name: t("common.slack"),
      type: Webhook_Type.SLACK,
      enabled: state.setting.slack?.enabled,
      render: () => {
        return (
          <div>
            <div class="textlabel">Token</div>
            <BBTextField
              class="mt-2"
              placeholder={t("common.write-only")}
              value={state.setting.slack?.token ?? ""}
              onUpdate:value={(val: string) => {
                state.setting.slack!.token = val;
              }}
            />
          </div>
        );
      },
    },
    {
      name: t("common.feishu"),
      type: Webhook_Type.FEISHU,
      enabled: state.setting.feishu?.enabled,
      render: () => {
        return (
          <div class="space-y-4">
            <div>
              <div class="textlabel">App ID</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.feishu?.appId ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.feishu!.appId = val;
                }}
              />
            </div>
            <div>
              <div class="textlabel">App Secret</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.feishu?.appSecret ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.feishu!.appSecret = val;
                }}
              />
            </div>
          </div>
        );
      },
    },
    {
      name: t("common.lark"),
      type: Webhook_Type.LARK,
      enabled: state.setting.lark?.enabled,
      render: () => {
        return (
          <div class="space-y-4">
            <div>
              <div class="textlabel">App ID</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.lark?.appId ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.lark!.appId = val;
                }}
              />
            </div>
            <div>
              <div class="textlabel">App Secret</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.lark?.appSecret ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.lark!.appSecret = val;
                }}
              />
            </div>
          </div>
        );
      },
    },
    {
      name: t("common.wecom"),
      type: Webhook_Type.WECOM,
      enabled: state.setting.wecom?.enabled,
      render: () => {
        return (
          <div class="space-y-4">
            <div>
              <div class="textlabel">Corp ID</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.wecom?.corpId ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.wecom!.corpId = val;
                }}
              />
            </div>
            <div>
              <div class="textlabel">Agent ID</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.wecom?.agentId ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.wecom!.agentId = val;
                }}
              />
            </div>
            <div>
              <div class="textlabel">Secret</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.wecom?.secret ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.wecom!.secret = val;
                }}
              />
            </div>
          </div>
        );
      },
    },
    {
      name: t("common.dingtalk"),
      type: Webhook_Type.DINGTALK,
      enabled: state.setting.dingtalk?.enabled,
      render: () => {
        return (
          <div class="space-y-4">
            <div>
              <div class="textlabel">Client ID</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.dingtalk?.clientId ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.dingtalk!.clientId = val;
                }}
              />
            </div>
            <div>
              <div class="textlabel">Client Secret</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.dingtalk?.clientSecret ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.dingtalk!.clientSecret = val;
                }}
              />
            </div>
            <div>
              <div class="textlabel">Robot Code</div>
              <BBTextField
                class="mt-2"
                placeholder={t("common.write-only")}
                value={state.setting.dingtalk?.robotCode ?? ""}
                onUpdate:value={(val: string) => {
                  state.setting.dingtalk!.robotCode = val;
                }}
              />
            </div>
          </div>
        );
      },
    },
  ];
});

const dataChanged = computed(() => {
  switch (state.selectedTab) {
    case Webhook_Type.SLACK:
      return !isEqual(
        state.setting.slack,
        imSetting.value.slack ?? AppIMSetting_Slack.fromPartial({})
      );
    case Webhook_Type.FEISHU:
      return !isEqual(
        state.setting.feishu,
        imSetting.value.feishu ?? AppIMSetting_Feishu.fromPartial({})
      );
    case Webhook_Type.WECOM:
      return !isEqual(
        state.setting.wecom,
        imSetting.value.wecom ?? AppIMSetting_Wecom.fromPartial({})
      );
    case Webhook_Type.LARK:
      return !isEqual(
        state.setting.lark,
        imSetting.value.lark ?? AppIMSetting_Lark.fromPartial({})
      );
    case Webhook_Type.DINGTALK:
      return !isEqual(
        state.setting.dingtalk,
        imSetting.value.dingtalk ?? AppIMSetting_DingTalk.fromPartial({})
      );
    default:
      return false;
  }
});

const canSave = computed(() => {
  switch (state.selectedTab) {
    case Webhook_Type.SLACK:
      return !!state.setting.slack?.token;
    case Webhook_Type.FEISHU:
      return !!state.setting.feishu?.appId && !!state.setting.feishu?.appSecret;
    case Webhook_Type.WECOM:
      return (
        !!state.setting.wecom?.corpId &&
        !!state.setting.wecom?.agentId &&
        !!state.setting.wecom?.secret
      );
    case Webhook_Type.LARK:
      return !!state.setting.lark?.appId && !!state.setting.lark?.appSecret;
    case Webhook_Type.DINGTALK:
      return (
        !!state.setting.dingtalk?.clientId &&
        !!state.setting.dingtalk?.clientSecret &&
        !!state.setting.dingtalk?.robotCode
      );
    default:
      return false;
  }
});

const discardChanges = () => {
  switch (state.selectedTab) {
    case Webhook_Type.SLACK:
      state.setting.slack = AppIMSetting_Slack.fromPartial({});
      break;
    case Webhook_Type.FEISHU:
      state.setting.feishu = AppIMSetting_Feishu.fromPartial({});
      break;
    case Webhook_Type.WECOM:
      state.setting.wecom = AppIMSetting_Wecom.fromPartial({});
      break;
    case Webhook_Type.LARK:
      state.setting.lark = AppIMSetting_Lark.fromPartial({});
    case Webhook_Type.DINGTALK:
      state.setting.dingtalk = AppIMSetting_DingTalk.fromPartial({});
      break;
  }
};

const onSave = async () => {
  state.loading = true;
  const updateMask: string[] = [];
  const data = cloneDeep(state.setting);

  switch (state.selectedTab) {
    case Webhook_Type.SLACK:
      updateMask.push("value.app_im_setting_value.slack");
      data.slack!.enabled = true;
      break;
    case Webhook_Type.FEISHU:
      updateMask.push("value.app_im_setting_value.feishu");
      data.feishu!.enabled = true;
      break;
    case Webhook_Type.WECOM:
      updateMask.push("value.app_im_setting_value.wecom");
      data.wecom!.enabled = true;
      break;
    case Webhook_Type.LARK:
      updateMask.push("value.app_im_setting_value.lark");
      data.lark!.enabled = true;
      break;
    case Webhook_Type.DINGTALK:
      updateMask.push("value.app_im_setting_value.dingtalk");
      data.dingtalk!.enabled = true;
      break;
  }

  try {
    const setting = await settingStore.upsertSetting({
      name: Setting_SettingName.APP_IM,
      value: {
        appImSettingValue: data,
      },
      updateMask,
    });

    switch (state.selectedTab) {
      case Webhook_Type.SLACK:
        state.setting.slack =
          setting.value?.appImSettingValue?.slack ??
          AppIMSetting_Slack.fromPartial({});
        break;
      case Webhook_Type.FEISHU:
        state.setting.feishu =
          setting.value?.appImSettingValue?.feishu ??
          AppIMSetting_Feishu.fromPartial({});
        break;
      case Webhook_Type.WECOM:
        state.setting.wecom =
          setting.value?.appImSettingValue?.wecom ??
          AppIMSetting_Wecom.fromPartial({});
        break;
      case Webhook_Type.LARK:
        state.setting.lark =
          setting.value?.appImSettingValue?.lark ??
          AppIMSetting_Lark.fromPartial({});
        break;
      case Webhook_Type.DINGTALK:
        state.setting.dingtalk =
          setting.value?.appImSettingValue?.dingtalk ??
          AppIMSetting_DingTalk.fromPartial({});
        break;
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.loading = false;
  }
};
</script>
