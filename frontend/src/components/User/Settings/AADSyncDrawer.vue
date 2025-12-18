<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent
      class="w-200 max-w-[90vw] relative"
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

          <MissingExternalURLAttention />

          <div class="flex flex-col gap-y-2">
            <div class="gap-x-2">
              <div class="font-medium">
                {{ $t(`settings.members.entra-sync.endpoint`) }}
              </div>
              <div class="text-sm text-gray-400">
                {{ $t(`settings.members.entra-sync.endpoint-tip`) }}
              </div>
            </div>
            <div class="flex gap-x-2">
              <NInput
                ref="scimUrlFieldRef"
                class="w-full"
                readonly
                :value="scimUrl"
                :placeholder="$t('banner.external-url')"
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

          <div class="flex flex-col gap-y-2">
            <div class="gap-x-2">
              <div class="font-medium">
                {{ $t(`settings.members.entra-sync.secret-token`) }}
              </div>
              <div class="text-sm text-gray-400">
                {{ $t("settings.members.entra-sync.secret-token-tip") }}
              </div>
            </div>
            <div class="flex gap-x-2">
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
            <div v-if="hasPermission">
              <NButton
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
        </div>
      </template>
      <template #footer>
        <div class="flex flex-row items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { ReplyIcon } from "lucide-vue-next";
import { NButton, NInput, useDialog } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { CopyButton, Drawer, DrawerContent } from "@/components/v2";
import { MissingExternalURLAttention } from "@/components/v2/Form";
import {
  pushNotification,
  useActuatorV1Store,
  useSettingV1Store,
} from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

defineProps<{
  show: boolean;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const settingV1Store = useSettingV1Store();
const actuatorStore = useActuatorV1Store();
const { t } = useI18n();
const scimUrlFieldRef = ref<HTMLInputElement | null>(null);
const scimTokenFieldRef = ref<HTMLInputElement | null>(null);
const $dialog = useDialog();

const hasPermission = computed(() =>
  hasWorkspacePermissionV2("bb.settings.set")
);

const workspaceId = computed(() => {
  return actuatorStore.serverInfo?.workspaceId ?? "";
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
  return settingV1Store.workspaceProfileSetting?.directorySyncToken ?? "";
});

const handleSelect = (component: HTMLInputElement | null) => {
  component?.select();
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
    onPositiveClick: async () => {
      await settingV1Store.updateWorkspaceProfile({
        payload: {
          directorySyncToken: "",
        },
        updateMask: create(FieldMaskSchema, {
          paths: ["value.workspace_profile.directory_sync_token"],
        }),
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    },
  });
};
</script>
