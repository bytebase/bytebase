<template>
  <div class="w-full flex flex-col gap-y-4 pb-4">
    <div class="textinfolabel">
      {{ $t("settings.im-integration.description") }}
      <LearnMoreLink
        url="https://docs.bytebase.com/change-database/webhook?source=console"
      />
    </div>

    <NEmpty
      v-if="state.setting.settings.length === 0"
      class="py-12 border rounded-sm"
    >
      <template #extra>
        <NButton
          v-if="availableImSettings.length > 0"
          type="primary"
          @click="() => onAddIM(availableImSettings[0].value)"
        >
          {{ $t("settings.im.add-im-integration") }}
        </NButton>
      </template>
    </NEmpty>
    <div v-else class="flex flex-col gap-y-4">
      <div
        v-for="(item, i) in state.setting.settings"
        :key="item.type"
        class="border rounded-sm p-4"
      >
        <template v-if="isConfigured(item.type)">
          <component :is="renderOption({ value: item.type })" />
        </template>
        <NSelect
          v-else
          v-model:value="item.type"
          :options="availableImSettings"
          :render-label="renderOption"
        />

        <div class="mt-4">
          <div v-if="item.type === Webhook_Type.SLACK">
            <div class="textlabel">Token</div>
            <BBTextField
              class="mt-2"
              :disabled="!props.allowEdit"
              :placeholder="t('common.sensitive-placeholder')"
              v-model:value="(item.payload.value as AppIMSetting_Slack).token"
            />
          </div>
          <div
            v-else-if="item.type === Webhook_Type.FEISHU"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">App ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Feishu).appId"
              />
            </div>
            <div>
              <div class="textlabel">App Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Feishu).appSecret"
              />
            </div>
          </div>
          <div
            v-else-if="item.type === Webhook_Type.WECOM"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">Corp ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Wecom).corpId"
              />
            </div>
            <div>
              <div class="textlabel">Agent ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Wecom).agentId"
              />
            </div>
            <div>
              <div class="textlabel">Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Wecom).secret"
              />
            </div>
          </div>
          <div
            v-else-if="item.type === Webhook_Type.LARK"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">App ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Lark).appId"
              />
            </div>
            <div>
              <div class="textlabel">App Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Lark).appSecret"
              />
            </div>
          </div>
          <div
            v-else-if="item.type === Webhook_Type.DINGTALK"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">Client ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_DingTalk).clientId"
              />
            </div>
            <div>
              <div class="textlabel">Client Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_DingTalk).clientSecret"
              />
            </div>
            <div>
              <div class="textlabel">Robot Code</div>
              <BBTextField
                class="mt-2"
                :disabled="!props.allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_DingTalk).robotCode"
              />
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between mt-4 gap-x-2">
          <div>
            <NPopconfirm
              v-if="isConfigured(item.type)"
              :positive-button-props="{
                type: 'error',
              }"
              @positive-click="() => onDeleteIM(i, item.type)"
            >
              <template #trigger>
                <NButton quaternary circle @click.stop type="error">
                  <template #icon>
                    <Trash2Icon class="w-4 h-auto" />
                  </template>
                </NButton>
              </template>
              <template #default>
                {{ $t("bbkit.confirm-button.sure-to-delete") }}
              </template>
            </NPopconfirm>
          </div>
          <div
            v-if="isDataChanged(item.type, item.payload.value)"
            class="flex items-center gap-x-2"
          >
            <NButton
              tertiary
              :disabled="!!state.pendingSaveType"
              @click="() => onDiscardIM(i, item.type)"
            >
              {{ $t("common.discard-changes") }}
            </NButton>
            <NButton
              type="primary"
              :disabled="!!state.pendingSaveType"
              :loading="state.pendingSaveType === item.type"
              @click="() => onSaveIM(i, item.type)"
            >
              {{ $t("common.save") }}
            </NButton>
          </div>
        </div>
      </div>

      <div class="flex justify-end">
        <NButton
          v-if="availableImSettings.length > 0"
          type="primary"
          secondary
          @click="() => onAddIM(availableImSettings[0].value)"
        >
          {{ $t("settings.im.add-another-im") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { Trash2Icon } from "lucide-vue-next";
import { NButton, NEmpty, NPopconfirm, NSelect } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import BBTextField from "@/bbkit/BBTextField.vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import WebhookTypeIcon from "@/components/Project/WebhookTypeIcon.vue";
import { pushNotification, useSettingV1Store } from "@/store";
import { Webhook_Type } from "@/types/proto-es/v1/project_service_pb";
import {
  type AppIMSetting,
  type AppIMSetting_DingTalk,
  AppIMSetting_DingTalkSchema,
  type AppIMSetting_Feishu,
  AppIMSetting_FeishuSchema,
  type AppIMSetting_IMSetting,
  AppIMSetting_IMSettingSchema,
  type AppIMSetting_Lark,
  AppIMSetting_LarkSchema,
  type AppIMSetting_Slack,
  AppIMSetting_SlackSchema,
  type AppIMSetting_Wecom,
  AppIMSetting_WecomSchema,
  AppIMSettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";

interface LocalState {
  setting: AppIMSetting;
  pendingSaveType?: Webhook_Type;
}

type IMSettingPayloadValue =
  | AppIMSetting_Slack
  | AppIMSetting_Feishu
  | AppIMSetting_Wecom
  | AppIMSetting_Lark
  | AppIMSetting_DingTalk
  | undefined;

const props = defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  setting: create(AppIMSettingSchema, {
    settings: [],
  }),
});

const settingStore = useSettingV1Store();

const onAddIM = (type: Webhook_Type) => {
  let setting: AppIMSetting_IMSetting | undefined = undefined;
  switch (type) {
    case Webhook_Type.SLACK:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "slack",
          value: create(AppIMSetting_SlackSchema, {}),
        },
      });
      break;
    case Webhook_Type.FEISHU:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "feishu",
          value: create(AppIMSetting_FeishuSchema, {}),
        },
      });
      break;
    case Webhook_Type.LARK:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "lark",
          value: create(AppIMSetting_LarkSchema, {}),
        },
      });
      break;
    case Webhook_Type.WECOM:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "wecom",
          value: create(AppIMSetting_WecomSchema, {}),
        },
      });
      break;
    case Webhook_Type.DINGTALK:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "dingtalk",
          value: create(AppIMSetting_DingTalkSchema, {}),
        },
      });
      break;
  }
  if (!setting) {
    return;
  }
  state.setting.settings.push(setting);
};

const onDeleteIM = async (index: number, type: Webhook_Type) => {
  state.setting.settings.splice(index, 1);
  if (isConfigured(type)) {
    await settingStore.upsertSetting({
      name: Setting_SettingName.APP_IM,
      value: create(SettingValueSchema, {
        value: {
          case: "appIm",
          value: create(AppIMSettingSchema, {
            settings: imSetting.value.settings.filter(
              (setting) => setting.type !== type
            ),
          }),
        },
      }),
    });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
  }
};

const onDiscardIM = (index: number, type: Webhook_Type) => {
  const oldSetting = imSetting.value.settings.find(
    (setting) => setting.type === type
  );
  if (oldSetting) {
    state.setting.settings[index] = cloneDeep(oldSetting);
  } else {
    state.setting.settings.splice(index, 1);
  }
};

const onSaveIM = async (index: number, type: Webhook_Type) => {
  state.pendingSaveType = type;

  const updateMask: string[] = [];
  switch (type) {
    case Webhook_Type.SLACK:
      updateMask.push("value.app_im.slack");
      break;
    case Webhook_Type.FEISHU:
      updateMask.push("value.app_im.feishu");
      break;
    case Webhook_Type.WECOM:
      updateMask.push("value.app_im.wecom");
      break;
    case Webhook_Type.LARK:
      updateMask.push("value.app_im.lark");
      break;
    case Webhook_Type.DINGTALK:
      updateMask.push("value.app_im.dingtalk");
      break;
  }

  const setting = state.setting.settings[index];
  const oldIndex = imSetting.value.settings.findIndex(
    (setting) => setting.type === type
  );
  const pendingUpdate = cloneDeep(imSetting.value);
  if (oldIndex < 0) {
    pendingUpdate.settings.push(setting);
  } else {
    pendingUpdate.settings[oldIndex] = setting;
  }
  try {
    await settingStore.upsertSetting({
      name: Setting_SettingName.APP_IM,
      value: create(SettingValueSchema, {
        value: {
          case: "appIm",
          value: pendingUpdate,
        },
      }),
      updateMask: create(FieldMaskSchema, { paths: updateMask }),
    });
    onDiscardIM(index, type);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.pendingSaveType = undefined;
  }
};

const fillMask = (data: IMSettingPayloadValue): IMSettingPayloadValue => {
  if (!data) {
    return data;
  }
  const result = { ...data };
  for (const key of Object.keys(result)) {
    if (key === "$typeName" || key === "$unknown") {
      continue;
    }
    const k = key as keyof IMSettingPayloadValue;
    if (typeof result[k] === "string" && result[k] === "") {
      (result[k] as string) = "*********";
    }
  }
  return result;
};

const imSetting = computed(() => {
  const setting = settingStore.getSettingByName(Setting_SettingName.APP_IM);
  if (setting?.value?.value?.case !== "appIm") {
    return create(AppIMSettingSchema, {
      settings: [],
    });
  }
  return create(AppIMSettingSchema, {
    settings: setting.value.value.value.settings.map((imSetting) => {
      return {
        ...imSetting,
        payload: {
          case: imSetting.payload.case,
          value: imSetting.payload.value
            ? fillMask(imSetting.payload.value)
            : undefined,
        },
      } as AppIMSetting_IMSetting;
    }),
  });
});

const isConfigured = (type: Webhook_Type) => {
  return (
    imSetting.value.settings.findIndex((setting) => setting.type === type) >= 0
  );
};

const isExisted = (type: Webhook_Type) => {
  return (
    state.setting.settings.findIndex((setting) => setting.type === type) >= 0
  );
};

const getImLabel = (type: Webhook_Type): string => {
  switch (type) {
    case Webhook_Type.SLACK:
      return t("common.slack");
    case Webhook_Type.FEISHU:
      return t("common.feishu");
    case Webhook_Type.LARK:
      return t("common.lark");
    case Webhook_Type.WECOM:
      return t("common.wecom");
    case Webhook_Type.DINGTALK:
      return t("common.dingtalk");
    default:
      return "";
  }
};

const availableImSettings = computed(() => {
  return [
    Webhook_Type.SLACK,
    Webhook_Type.FEISHU,
    Webhook_Type.LARK,
    Webhook_Type.WECOM,
    Webhook_Type.DINGTALK,
  ]
    .filter((type) => !isConfigured(type) && !isExisted(type))
    .map((type) => ({
      value: type,
    }));
});

const renderOption = ({ value }: { value: Webhook_Type }) => {
  return (
    <div class="flex items-center gap-x-2">
      <WebhookTypeIcon type={value} class="h-5 w-5" />
      {getImLabel(value)}
    </div>
  );
};

watch(
  () => imSetting.value,
  (setting) => {
    state.setting = cloneDeep(setting);
  },
  { once: true, immediate: true }
);

const isDataChanged = (
  type: Webhook_Type,
  value: IMSettingPayloadValue
): boolean => {
  const oldSetting = imSetting.value.settings.find(
    (setting) => setting.type === type
  )?.payload.value;
  return !isEqual(value, oldSetting);
};
</script>
