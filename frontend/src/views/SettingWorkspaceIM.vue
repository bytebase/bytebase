<template>
  <div class="w-full space-y-4">
    <div class="textinfolabel">
      {{ $t("settings.im-integration.description") }}
      <a
        class="normal-link inline-flex items-center"
        href="https://www.bytebase.com/docs/change-database/webhook?source=console"
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
  AppIMSetting_Wecom,
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
  selectedTab: Webhook_Type.TYPE_SLACK,
  loading: false,
  setting: AppIMSetting.fromPartial({}),
});

const settingStore = useSettingV1Store();

const imSetting = computed(
  () =>
    settingStore.getSettingByName("bb.app.im")?.value?.appImSettingValue ??
    AppIMSetting.fromPartial({})
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
      case Webhook_Type.TYPE_SLACK:
        if (!state.setting.slack) {
          state.setting.slack = AppIMSetting_Slack.fromPartial({});
        }
        break;
      case Webhook_Type.TYPE_DINGTALK:
        if (!state.setting.feishu) {
          state.setting.feishu = AppIMSetting_Feishu.fromPartial({});
        }
        break;
      case Webhook_Type.TYPE_WECOM:
        if (!state.setting.wecom) {
          state.setting.wecom = AppIMSetting_Wecom.fromPartial({});
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
      type: Webhook_Type.TYPE_SLACK,
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
      type: Webhook_Type.TYPE_FEISHU,
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
      name: t("common.wecom"),
      type: Webhook_Type.TYPE_WECOM,
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
  ];
});

const dataChanged = computed(() => {
  switch (state.selectedTab) {
    case Webhook_Type.TYPE_SLACK:
      return !isEqual(
        state.setting.slack,
        imSetting.value.slack ?? AppIMSetting_Slack.fromPartial({})
      );
    case Webhook_Type.TYPE_DINGTALK:
      return !isEqual(
        state.setting.feishu,
        imSetting.value.feishu ?? AppIMSetting_Feishu.fromPartial({})
      );
    case Webhook_Type.TYPE_WECOM:
      return !isEqual(
        state.setting.wecom,
        imSetting.value.wecom ?? AppIMSetting_Wecom.fromPartial({})
      );
    default:
      return false;
  }
});

const canSave = computed(() => {
  switch (state.selectedTab) {
    case Webhook_Type.TYPE_SLACK:
      return !!state.setting.slack?.token;
    case Webhook_Type.TYPE_DINGTALK:
      return !!state.setting.feishu?.appId && !!state.setting.feishu?.appSecret;
    case Webhook_Type.TYPE_WECOM:
      return (
        !!state.setting.wecom?.corpId &&
        !!state.setting.wecom?.agentId &&
        !!state.setting.wecom?.secret
      );
    default:
      return false;
  }
});

const discardChanges = () => {
  switch (state.selectedTab) {
    case Webhook_Type.TYPE_SLACK:
      state.setting.slack = AppIMSetting_Slack.fromPartial({});
      break;
    case Webhook_Type.TYPE_DINGTALK:
      state.setting.feishu = AppIMSetting_Feishu.fromPartial({});
      break;
    case Webhook_Type.TYPE_WECOM:
      state.setting.wecom = AppIMSetting_Wecom.fromPartial({});
      break;
  }
};

const onSave = async () => {
  state.loading = true;
  const updateMask: string[] = [];
  const data = cloneDeep(state.setting);

  switch (state.selectedTab) {
    case Webhook_Type.TYPE_SLACK:
      updateMask.push("value.app_im_setting_value.slack");
      data.slack!.enabled = true;
      break;
    case Webhook_Type.TYPE_DINGTALK:
      updateMask.push("value.app_im_setting_value.feishu");
      data.feishu!.enabled = true;
      break;
    case Webhook_Type.TYPE_WECOM:
      updateMask.push("value.app_im_setting_value.wecom");
      data.wecom!.enabled = true;
      break;
  }

  try {
    const setting = await settingStore.upsertSetting({
      name: "bb.app.im",
      value: {
        appImSettingValue: data,
      },
      updateMask,
    });

    switch (state.selectedTab) {
      case Webhook_Type.TYPE_SLACK:
        state.setting.slack =
          setting.value?.appImSettingValue?.slack ??
          AppIMSetting_Slack.fromPartial({});
        break;
      case Webhook_Type.TYPE_DINGTALK:
        state.setting.feishu =
          setting.value?.appImSettingValue?.feishu ??
          AppIMSetting_Feishu.fromPartial({});
        break;
      case Webhook_Type.TYPE_WECOM:
        state.setting.wecom =
          setting.value?.appImSettingValue?.wecom ??
          AppIMSetting_Wecom.fromPartial({});
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
