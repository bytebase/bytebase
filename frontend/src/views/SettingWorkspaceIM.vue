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
        <PermissionGuardWrapper
          v-if="availableImSettings.length > 0"
          v-slot="slotProps"
          :permissions="['bb.settings.set']"
        >
          <NButton
            type="primary"
            :disabled="slotProps.disabled"
            @click="() => onAddIM(availableImSettings[0].value)"
          >
            <template #icon>
              <PlusIcon class="h-4 w-4" />
            </template>
            {{ $t("settings.im.add-im-integration") }}
          </NButton>
        </PermissionGuardWrapper>
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
          <div v-if="item.type === WebhookType.SLACK">
            <div class="textlabel">Token</div>
            <BBTextField
              class="mt-2"
              :disabled="!allowEdit"
              :placeholder="t('common.sensitive-placeholder')"
              v-model:value="(item.payload.value as AppIMSetting_Slack).token"
            />
          </div>
          <div
            v-else-if="item.type === WebhookType.FEISHU"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">App ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Feishu).appId"
              />
            </div>
            <div>
              <div class="textlabel">App Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Feishu).appSecret"
              />
            </div>
          </div>
          <div
            v-else-if="item.type === WebhookType.WECOM"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">Corp ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Wecom).corpId"
              />
            </div>
            <div>
              <div class="textlabel">Agent ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Wecom).agentId"
              />
            </div>
            <div>
              <div class="textlabel">Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Wecom).secret"
              />
            </div>
          </div>
          <div
            v-else-if="item.type === WebhookType.LARK"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">App ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Lark).appId"
              />
            </div>
            <div>
              <div class="textlabel">App Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Lark).appSecret"
              />
            </div>
          </div>
          <div
            v-else-if="item.type === WebhookType.DINGTALK"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">Client ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_DingTalk).clientId"
              />
            </div>
            <div>
              <div class="textlabel">Client Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_DingTalk).clientSecret"
              />
            </div>
            <div>
              <div class="textlabel">Robot Code</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_DingTalk).robotCode"
              />
            </div>
          </div>
          <div
            v-else-if="item.type === WebhookType.TEAMS"
            class="flex flex-col gap-y-4"
          >
            <div>
              <div class="textlabel">Tenant ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Teams).tenantId"
              />
            </div>
            <div>
              <div class="textlabel">Client ID</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Teams).clientId"
              />
            </div>
            <div>
              <div class="textlabel">Client Secret</div>
              <BBTextField
                class="mt-2"
                :disabled="!allowEdit"
                :placeholder="t('common.sensitive-placeholder')"
                v-model:value="(item.payload.value as AppIMSetting_Teams).clientSecret"
              />
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between mt-4 gap-x-2">
          <div>
            <NPopconfirm
              v-if="isConfigured(item.type) && allowEdit"
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
              :disabled="!!state.pendingSaveType || !allowEdit"
              :loading="state.pendingSaveType === item.type"
              @click="() => onSaveIM(i, item.type)"
            >
              {{ $t("common.save") }}
            </NButton>
          </div>
        </div>
      </div>

      <div v-if="availableImSettings.length > 0" class="flex justify-end">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="['bb.settings.set']"
        >
          <NButton
            type="primary"
            secondary
            :disabled="slotProps.disabled"
            @click="() => onAddIM(availableImSettings[0].value)"
          >
            <template #icon>
              <PlusIcon class="h-4 w-4" />
            </template>
            {{ $t("settings.im.add-another-im") }}
          </NButton>
        </PermissionGuardWrapper>
      </div>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { clone, create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { PlusIcon, Trash2Icon } from "lucide-vue-next";
import { NButton, NEmpty, NPopconfirm, NSelect } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import BBTextField from "@/bbkit/BBTextField.vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import WebhookTypeIcon from "@/components/Project/WebhookTypeIcon.vue";
import { pushNotification, useSettingV1Store } from "@/store";
import { WebhookType } from "@/types/proto-es/v1/common_pb";
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
  type AppIMSetting_Teams,
  AppIMSetting_TeamsSchema,
  type AppIMSetting_Wecom,
  AppIMSetting_WecomSchema,
  AppIMSettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";

interface LocalState {
  setting: AppIMSetting;
  pendingSaveType?: WebhookType;
}

type IMSettingPayloadValue =
  | AppIMSetting_Slack
  | AppIMSetting_Feishu
  | AppIMSetting_Wecom
  | AppIMSetting_Lark
  | AppIMSetting_DingTalk
  | AppIMSetting_Teams
  | undefined;

defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  setting: create(AppIMSettingSchema, {
    settings: [],
  }),
});

const settingStore = useSettingV1Store();

const onAddIM = (type: WebhookType) => {
  let setting: AppIMSetting_IMSetting | undefined = undefined;
  switch (type) {
    case WebhookType.SLACK:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "slack",
          value: create(AppIMSetting_SlackSchema, {}),
        },
      });
      break;
    case WebhookType.FEISHU:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "feishu",
          value: create(AppIMSetting_FeishuSchema, {}),
        },
      });
      break;
    case WebhookType.LARK:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "lark",
          value: create(AppIMSetting_LarkSchema, {}),
        },
      });
      break;
    case WebhookType.WECOM:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "wecom",
          value: create(AppIMSetting_WecomSchema, {}),
        },
      });
      break;
    case WebhookType.DINGTALK:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "dingtalk",
          value: create(AppIMSetting_DingTalkSchema, {}),
        },
      });
      break;
    case WebhookType.TEAMS:
      setting = create(AppIMSetting_IMSettingSchema, {
        type,
        payload: {
          case: "teams",
          value: create(AppIMSetting_TeamsSchema, {}),
        },
      });
      break;
  }
  if (!setting) {
    return;
  }
  state.setting.settings.push(setting);
};

const onDeleteIM = async (index: number, type: WebhookType) => {
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

const onDiscardIM = (index: number, type: WebhookType) => {
  const oldSetting = imSetting.value.settings.find(
    (setting) => setting.type === type
  );
  if (oldSetting) {
    state.setting.settings[index] = clone(
      AppIMSetting_IMSettingSchema,
      oldSetting
    );
  } else {
    state.setting.settings.splice(index, 1);
  }
};

const onSaveIM = async (index: number, type: WebhookType) => {
  state.pendingSaveType = type;

  const updateMask: string[] = [];
  switch (type) {
    case WebhookType.SLACK:
      updateMask.push("value.app_im.slack");
      break;
    case WebhookType.FEISHU:
      updateMask.push("value.app_im.feishu");
      break;
    case WebhookType.WECOM:
      updateMask.push("value.app_im.wecom");
      break;
    case WebhookType.LARK:
      updateMask.push("value.app_im.lark");
      break;
    case WebhookType.DINGTALK:
      updateMask.push("value.app_im.dingtalk");
      break;
    case WebhookType.TEAMS:
      updateMask.push("value.app_im_setting_value.teams");
      break;
  }

  const setting = state.setting.settings[index];
  const oldIndex = imSetting.value.settings.findIndex(
    (setting) => setting.type === type
  );

  // Reconstruct the setting with proper oneof payload to avoid Vue reactivity issues
  const reconstructSetting = (): AppIMSetting_IMSetting => {
    const payloadValue = setting.payload.value;
    switch (type) {
      case WebhookType.SLACK:
        return create(AppIMSetting_IMSettingSchema, {
          type,
          payload: {
            case: "slack",
            value: create(
              AppIMSetting_SlackSchema,
              payloadValue as AppIMSetting_Slack
            ),
          },
        });
      case WebhookType.FEISHU:
        return create(AppIMSetting_IMSettingSchema, {
          type,
          payload: {
            case: "feishu",
            value: create(
              AppIMSetting_FeishuSchema,
              payloadValue as AppIMSetting_Feishu
            ),
          },
        });
      case WebhookType.WECOM:
        return create(AppIMSetting_IMSettingSchema, {
          type,
          payload: {
            case: "wecom",
            value: create(
              AppIMSetting_WecomSchema,
              payloadValue as AppIMSetting_Wecom
            ),
          },
        });
      case WebhookType.LARK:
        return create(AppIMSetting_IMSettingSchema, {
          type,
          payload: {
            case: "lark",
            value: create(
              AppIMSetting_LarkSchema,
              payloadValue as AppIMSetting_Lark
            ),
          },
        });
      case WebhookType.DINGTALK:
        return create(AppIMSetting_IMSettingSchema, {
          type,
          payload: {
            case: "dingtalk",
            value: create(
              AppIMSetting_DingTalkSchema,
              payloadValue as AppIMSetting_DingTalk
            ),
          },
        });
      case WebhookType.TEAMS:
        return create(AppIMSetting_IMSettingSchema, {
          type,
          payload: {
            case: "teams",
            value: create(
              AppIMSetting_TeamsSchema,
              payloadValue as AppIMSetting_Teams
            ),
          },
        });
      default:
        return setting;
    }
  };

  const reconstructedSetting = reconstructSetting();

  // Use protobuf clone to properly preserve oneof discriminators
  const pendingUpdate = clone(AppIMSettingSchema, imSetting.value);
  if (oldIndex < 0) {
    pendingUpdate.settings.push(reconstructedSetting);
  } else {
    pendingUpdate.settings[oldIndex] = reconstructedSetting;
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

const isConfigured = (type: WebhookType) => {
  return (
    imSetting.value.settings.findIndex((setting) => setting.type === type) >= 0
  );
};

const isExisted = (type: WebhookType) => {
  return (
    state.setting.settings.findIndex((setting) => setting.type === type) >= 0
  );
};

const getImLabel = (type: WebhookType): string => {
  switch (type) {
    case WebhookType.SLACK:
      return t("common.slack");
    case WebhookType.FEISHU:
      return t("common.feishu");
    case WebhookType.LARK:
      return t("common.lark");
    case WebhookType.WECOM:
      return t("common.wecom");
    case WebhookType.DINGTALK:
      return t("common.dingtalk");
    case WebhookType.TEAMS:
      return t("common.teams");
    default:
      return "";
  }
};

const availableImSettings = computed(() => {
  return [
    WebhookType.SLACK,
    WebhookType.FEISHU,
    WebhookType.LARK,
    WebhookType.WECOM,
    WebhookType.DINGTALK,
    WebhookType.TEAMS,
  ]
    .filter((type) => !isConfigured(type) && !isExisted(type))
    .map((type) => ({
      value: type,
    }));
});

const renderOption = ({ value }: { value: WebhookType }) => {
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
    state.setting = clone(AppIMSettingSchema, setting);
  },
  { once: true, immediate: true }
);

const isDataChanged = (
  type: WebhookType,
  value: IMSettingPayloadValue
): boolean => {
  const oldSetting = imSetting.value.settings.find(
    (setting) => setting.type === type
  )?.payload.value;
  return !isEqual(value, oldSetting);
};
</script>
