<template>
  <div id="branding" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.branding") }}
        </h1>
        <FeatureBadge feature="bb.feature.branding" />
      </div>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>
    <div class="flex-1 lg:px-4">
      <div class="mb-4 mt-4 lg:mt-0">
        <span class="font-medium">
          {{ $t("settings.general.workspace.logo") }}
        </span>
        <p class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.logo-aspect") }}
        </p>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <div
              class="flex justify-center border-2 border-gray-300 border-dashed rounded-md relative h-48"
            >
              <div
                class="w-full bg-no-repeat bg-contain bg-center rounded-md pointer-events-none m-4"
                :style="`background-image: url(${state.logoUrl});`"
              ></div>
              <SingleFileSelector
                class="space-y-1 text-center flex flex-col justify-center items-center absolute top-0 bottom-0 left-0 right-0"
                :class="[state.logoUrl ? 'opacity-0 hover:opacity-90' : '']"
                :max-file-size-in-mi-b="maxFileSizeInMiB"
                :support-file-extensions="supportImageExtensions"
                :disabled="!allowEdit"
                @on-select="onLogoSelect"
              />
            </div>
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{ $t("settings.general.workspace.only-admin-can-edit") }}
          </span>
        </NTooltip>
      </div>
      <div class="flex justify-end gap-x-3">
        <NPopconfirm
          v-if="allowEdit && allowDelete"
          @positive-click="deleteLogo"
        >
          <template #trigger>
            <NButton :disabled="!allowEdit">
              {{ $t("common.delete") }}
            </NButton>
          </template>
          <template #default>
            {{ t("settings.general.workspace.confirm-delete-custom-logo") }}
          </template>
        </NPopconfirm>
        <NButton
          type="primary"
          :disabled="!allowEdit || !allowSave"
          @click.prevent="uploadLogo"
        >
          <FeatureBadge
            feature="bb.feature.branding"
            custom-class="mr-1 text-white pointer-events-none"
          />
          {{ $t("common.update") }}
        </NButton>
      </div>
    </div>
  </div>

  <FeatureModal
    feature="bb.feature.branding"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NButton, NPopconfirm, NTooltip } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { featureToRef, pushNotification } from "@/store";
import { useActuatorV1Store } from "@/store/modules/v1/actuator";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";
import SingleFileSelector from "../SingleFileSelector.vue";

interface LocalState {
  displayName?: string;
  logoUrl?: string;
  logoFile: File | null;
  loading: boolean;
  showFeatureModal: boolean;
}

defineProps<{
  allowEdit: boolean;
}>();

const maxFileSizeInMiB = 2;
const supportImageExtensions = [".jpg", ".jpeg", ".png", ".webp", ".svg"];

// convertFileToBase64 will convert a file into base64 string.
const convertFileToBase64 = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = (error) => reject(error);
  });

const settingV1Store = useSettingV1Store();
const { t } = useI18n();

const state = reactive<LocalState>({
  displayName: "",
  logoUrl: "",
  logoFile: null,
  loading: false,
  showFeatureModal: false,
});

watchEffect(() => {
  state.logoUrl = settingV1Store.brandingLogo;
});

const valid = computed((): boolean => {
  return !!state.displayName || !!state.logoFile;
});

const allowDelete = computed(() => {
  return settingV1Store.brandingLogo !== "";
});

const allowSave = computed((): boolean => {
  return state.logoFile !== null && valid.value && !state.loading;
});

const hasBrandingFeature = featureToRef("bb.feature.branding");

const doUpdate = async (content: string, message: string) => {
  state.loading = true;
  try {
    const setting = await settingV1Store.upsertSetting({
      name: "bb.branding.logo",
      value: {
        stringValue: content,
      },
    });

    useActuatorV1Store().setLogo(content);

    state.logoFile = null;
    state.logoUrl = setting.value?.stringValue;

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: message,
    });
  } finally {
    state.loading = false;
  }
};
const deleteLogo = async () => {
  state.loading = true;

  await doUpdate("", t("common.deleted"));
};

const uploadLogo = async () => {
  if (!hasBrandingFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  if (!state.logoFile) {
    return;
  }

  state.loading = true;
  const fileInBase64 = await convertFileToBase64(state.logoFile);

  await doUpdate(
    fileInBase64,
    t("settings.general.workspace.logo-upload-succeed")
  );
};

const onLogoSelect = (file: File) => {
  state.logoFile = file;
  state.logoUrl = URL.createObjectURL(file);
};
</script>
