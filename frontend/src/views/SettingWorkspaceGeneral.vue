<template>
  <div class="divide-y divide-block-border pt-2">
    <GeneralSetting
      ref="generalSettingRef"
      :title="$t('common.general')"
      :allow-edit="allowEdit"
    />
    <BrandingSetting
      ref="brandingSettingRef"
      :title="$t('settings.general.workspace.branding')"
      :allow-edit="allowEdit"
    />
    <AccountSetting
      ref="accountSettingRef"
      :title="$t('settings.general.workspace.account')"
      :allow-edit="allowEdit"
    />
    <SecuritySetting
      ref="securitySettingRef"
      :title="$t('settings.general.workspace.security')"
      :allow-edit="allowEdit"
    />
    <AIAugmentationSetting
      ref="aiAugmentationSettingRef"
      :title="$t('settings.general.workspace.ai-assistant.self')"
      :allow-edit="allowEdit"
    />
    <AnnouncementSetting
      ref="announcementSettingRef"
      :title="$t('settings.general.workspace.announcement.self')"
      :allow-edit="allowEdit"
    />
    <ProductImprovementSetting
      ref="productImprovementSettingRef"
      :allow-edit="allowEdit"
    />
    <AuditLogStdoutSetting
      ref="auditLogStdoutSettingRef"
      :allow-edit="allowEdit"
    />

    <div v-if="allowEdit && isDirty" class="sticky bottom-0 z-10">
      <div
        class="flex justify-between w-full py-4 border-t border-block-border bg-white"
      >
        <NButton @click.prevent="onRevert">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton type="primary" @click.prevent="onUpdate">
          {{ $t("common.confirm-and-update") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import {
  AccountSetting,
  AIAugmentationSetting,
  AnnouncementSetting,
  AuditLogStdoutSetting,
  BrandingSetting,
  GeneralSetting,
  ProductImprovementSetting,
  SecuritySetting,
} from "@/components/GeneralSetting";
import { useRouteChangeGuard } from "@/composables/useRouteChangeGuard";
import { useBodyLayoutContext } from "@/layouts/common";
import { pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";

defineProps<{
  allowEdit: boolean;
}>();

const route = useRoute();
const { t } = useI18n();

const accountSettingRef = ref<InstanceType<typeof AccountSetting>>();
const generalSettingRef = ref<InstanceType<typeof GeneralSetting>>();
const brandingSettingRef = ref<InstanceType<typeof BrandingSetting>>();
const securitySettingRef = ref<InstanceType<typeof SecuritySetting>>();
const aiAugmentationSettingRef =
  ref<InstanceType<typeof AIAugmentationSetting>>();
const announcementSettingRef = ref<InstanceType<typeof AnnouncementSetting>>();
const productImprovementSettingRef =
  ref<InstanceType<typeof ProductImprovementSetting>>();
const auditLogStdoutSettingRef =
  ref<InstanceType<typeof AuditLogStdoutSetting>>();

const settingRefList = computed(() => {
  return [
    accountSettingRef,
    generalSettingRef,
    brandingSettingRef,
    securitySettingRef,
    aiAugmentationSettingRef,
    announcementSettingRef,
    productImprovementSettingRef,
    auditLogStdoutSettingRef,
  ];
});

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("!pb-0");

onMounted(async () => {
  await useSettingV1Store().fetchSettingList();
  // If the route has a hash, try to scroll to the element with the value.
  if (route.hash) {
    document.body.querySelector(route.hash)?.scrollIntoView();
  }
});

const isDirty = computed(() => {
  return settingRefList.value.some((settingRef) => settingRef.value?.isDirty);
});

const onRevert = () => {
  for (const settingRef of settingRefList.value) {
    settingRef.value?.revert();
  }
};

const onUpdate = async () => {
  let failedCount = 0;
  let totalCount = 0;
  for (const settingRef of settingRefList.value) {
    if (settingRef.value?.isDirty) {
      totalCount++;
      try {
        await settingRef.value.update();
      } catch (e) {
        console.error(e); // for debug.
        failedCount++;
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: t("settings.general.workspace.failed-to-update-setting", {
            title: settingRef.value.title,
          }),
        });
      }
    }
  }
  if (totalCount > 0 && totalCount !== failedCount) {
    pushNotification({
      module: "bytebase",
      style: failedCount === 0 ? "SUCCESS" : "WARN",
      title:
        failedCount === 0
          ? t("settings.general.workspace.config-updated")
          : t("settings.general.workspace.config-partly-updated"),
    });
  }
};

useRouteChangeGuard(isDirty);
</script>
