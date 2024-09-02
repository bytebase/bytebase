<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent
      class="w-[50rem] max-w-[90vw] relative"
      :title="$t('settings.members.aad-sync.self')"
    >
      <template #default>
        <div class="flex flex-col gap-y-4">
          <div class="text-sm text-control-light">
            {{ $t(`settings.members.aad-sync.description`) }}
            <LearnMoreLink
              url="https://www.bytebase.com/docs/administration/directiry-sync?source=console"
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
                {{ $t(`settings.members.aad-sync.endpoint`) }}
              </div>
              <div class="text-sm text-gray-400">
                {{ $t(`settings.members.aad-sync.endpoint-tip`) }}
              </div>
            </div>
            <div class="flex space-x-2">
              <NInput
                ref="scimUrlFieldRef"
                class="w-full"
                readonly
                :value="scimUrl"
                @click="handleCopyUrl(scimUrlFieldRef)"
              />
              <NButton
                v-if="isSupported"
                :disabled="!scimUrl"
                @click="handleCopyUrl(scimUrlFieldRef)"
              >
                <heroicons-outline:clipboard-document class="w-4 h-4" />
              </NButton>
            </div>
          </div>

          <div class="space-y-2">
            <div class="gap-x-2">
              <div class="font-medium">
                {{ $t(`settings.members.aad-sync.token`) }}
              </div>
              <div class="text-sm text-gray-400">
                {{ $t(`settings.members.aad-sync.token-tip`) }}
              </div>
            </div>
            <div class="flex space-x-2">
              <NInput
                ref="scimTokenFieldRef"
                class="w-full"
                readonly
                type="password"
                :value="scimToken"
                @click="handleCopyUrl(scimTokenFieldRef)"
              />
              <NButton
                v-if="isSupported"
                :disabled="!scimToken"
                @click="handleCopyUrl(scimTokenFieldRef)"
              >
                <heroicons-outline:clipboard-document class="w-4 h-4" />
              </NButton>
            </div>
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
import { useClipboard } from "@vueuse/core";
import { NButton, NInput } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import { Drawer, DrawerContent } from "@/components/v2";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { pushNotification, useSettingV1Store } from "@/store";

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

const workspaceId = computed(() => {
  return (
    settingV1Store.getSettingByName("bb.workspace.id")?.value?.stringValue ?? ""
  );
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
  return (
    settingV1Store.getSettingByName("bb.workspace.scim")?.value?.scimSetting
      ?.token ?? ""
  );
});

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});

const handleCopyUrl = (component: HTMLInputElement | null) => {
  component?.select();
  const value = component?.value;
  if (!value) {
    return;
  }

  copyTextToClipboard(value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
};

const configureSetting = () => {
  router.push({
    name: SETTING_ROUTE_WORKSPACE_GENERAL,
  });
};
</script>
