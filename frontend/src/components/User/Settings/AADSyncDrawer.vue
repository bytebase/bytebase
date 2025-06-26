<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent
      class="w-[50rem] max-w-[90vw] relative"
      :title="$t('settings.members.entra-sync.self')"
    >
      <template #default>
        <div class="flex flex-col gap-y-4">
          <div class="text-sm text-control-light">
            {{ $t(`settings.members.entra-sync.description`) }}
            <LearnMoreLink
              url="https://docs.bytebase.com/administration/scim/overview?source=console"
              class="ml-1"
            />
          </div>

          <BBAttention
            v-if="!externalUrl"
            class="w-full border-none"
            type="error"
            :title="$t('banner.external-url')"
            :description="
              $t('settings.general.workspace.external-url.description')
            "
          >
            <template #action>
              <NButton type="primary" @click="configureSetting">
                {{ $t("common.configure-now") }}
              </NButton>
            </template>
          </BBAttention>

          <div class="space-y-2">
            <div class="gap-x-2">
              <div class="font-medium">
                {{ $t(`settings.members.entra-sync.endpoint`) }}
              </div>
              <div class="text-sm text-gray-400">
                {{ $t(`settings.members.entra-sync.endpoint-tip`) }}
              </div>
            </div>
            <div class="flex space-x-2">
              <NInput
                ref="scimUrlFieldRef"
                class="w-full"
                readonly
                :value="scimUrl"
                @click="handleSelect(scimUrlFieldRef)"
              />
              <CopyButton
                quaternary
                :text="false"
                :size="'medium'"
                :content="scimUrl"
                :disabled="!scimUrl"
                @click="handleSelect(scimUrlFieldRef)"
              />
            </div>
          </div>

          <div class="space-y-2">
            <div class="gap-x-2">
              <div class="font-medium">
                {{ $t(`settings.members.entra-sync.secret-token`) }}
              </div>
              <div class="text-sm text-gray-400">
                {{ $t("settings.members.entra-sync.secret-token-tip") }}
              </div>
            </div>
            <div class="flex space-x-2">
              <NInput
                ref="scimTokenFieldRef"
                class="w-full"
                readonly
                type="password"
                :value="scimToken"
                @click="handleSelect(scimTokenFieldRef)"
              />
              <CopyButton
                quaternary
                :text="false"
                :size="'medium'"
                :content="scimToken"
                :disabled="!scimToken"
                @click="handleSelect(scimTokenFieldRef)"
              />
            </div>
            <NButton
              v-if="hasPermission"
              tertiary
              :type="'warning'"
              :size="'small'"
              @click="resetToken"
            >
              <template #icon>
                <ReplyIcon class="w-4" />
              </template>
              {{ $t("settings.members.entra-sync.reset-token") }}
            </NButton>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex flex-row items-center justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { ReplyIcon } from "lucide-vue-next";
import { NButton, NInput, useDialog } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { CopyButton } from "@/components/v2";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { pushNotification, useSettingV1Store } from "@/store";
import { Setting_SettingName, SCIMSettingSchema, ValueSchema as SettingValueSchema } from "@/types/proto-es/v1/setting_service_pb";
import { create } from "@bufbuild/protobuf";
import { hasWorkspacePermissionV2 } from "@/utils";

defineProps<{
  show: boolean;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const settingV1Store = useSettingV1Store();
const { t } = useI18n();
const router = useRouter();
const scimUrlFieldRef = ref<HTMLInputElement | null>(null);
const scimTokenFieldRef = ref<HTMLInputElement | null>(null);
const $dialog = useDialog();

const hasPermission = computed(() =>
  hasWorkspacePermissionV2("bb.settings.set")
);

const workspaceId = computed(() => {
  const setting = settingV1Store.getSettingByName(Setting_SettingName.WORKSPACE_ID);
  return setting?.value?.value?.case === "stringValue" 
    ? setting.value.value.value 
    : "";
});

const externalUrl = computed(() => {
  return settingV1Store.workspaceProfileSetting?.externalUrl;
});

const scimUrl = computed(() => {
  if (!workspaceId.value || !externalUrl.value) {
    return "";
  }
  return `${externalUrl.value}/hook/scim/workspaces/${workspaceId.value}`;
});

const scimToken = computed(() => {
  const setting = settingV1Store.getSettingByName(Setting_SettingName.SCIM);
  return setting?.value?.value?.case === "scimSetting" 
    ? setting.value.value.value.token ?? "" 
    : "";
});

const handleSelect = (component: HTMLInputElement | null) => {
  component?.select();
};

const configureSetting = () => {
  router.push({
    name: SETTING_ROUTE_WORKSPACE_GENERAL,
  });
};

const resetToken = () => {
  $dialog.warning({
    title: t("common.warning"),
    style: "z-index: 100000",
    content: () => {
      return t("settings.members.entra-sync.reset-token-warning");
    },
    negativeText: t("common.cancel"),
    positiveText: t("common.continue-anyway"),
    closeOnEsc: true,
    maskClosable: true,
    onPositiveClick: () => {
      settingV1Store
        .upsertSetting({
          name: Setting_SettingName.SCIM,
          value: create(SettingValueSchema, {
            value: {
              case: "scimSetting",
              value: create(SCIMSettingSchema, {
                token: "",
              }),
            },
          }),
        })
        .then(() => {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("common.updated"),
          });
        });
    },
  });
};
</script>
