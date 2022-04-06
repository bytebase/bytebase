<template>
  <div class="mt-2 space-y-6 divide-y divide-block-border">
    <div class="px-4 py-6 lg:flex">
      <div class="text-left lg:w-1/4">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.branding") }}
        </h1>
        <span v-if="!allowEdit" class="text-sm text-gray-400">
          {{ $t("settings.general.workspace.only-owner-can-edit") }}
        </span>
      </div>
      <div class="flex-1 lg:px-5">
        <div class="mb-5 mt-5 lg:mt-0">
          <p>
            {{ $t("settings.general.workspace.logo") }}
          </p>
          <p class="mb-3 text-sm text-gray-400">
            {{ $t("settings.general.workspace.logo-aspect") }}
          </p>
          <div
            class="flex justify-center border-2 border-gray-300 border-dashed rounded-md relative h-48"
          >
            <div
              class="w-full bg-no-repeat bg-contain bg-center rounded-md pointer-events-none m-5"
              :style="`background-image: url(${state.logoUrl});`"
            ></div>
            <SingleFileSelector
              class="space-y-1 text-center flex flex-col justify-center items-center absolute top-0 bottom-0 left-0 right-0"
              :class="[state.logoUrl ? 'opacity-0 hover:opacity-90' : '']"
              :max-file-size-in-mi-b="maxFileSizeInMiB"
              :support-file-extensions="supportImageExtensions"
              @on-select="onLogoSelect"
            >
              <svg
                class="mx-auto h-12 w-12 text-gray-400 pointer-events-none"
                stroke="currentColor"
                fill="none"
                viewBox="0 0 48 48"
                aria-hidden="true"
              >
                <path
                  d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
              <div
                class="text-sm text-gray-600 inline-flex pointer-events-none"
              >
                <span
                  class="relative cursor-pointer rounded-md font-medium text-indigo-600 hover:text-indigo-500 focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2 focus-within:ring-indigo-500"
                >
                  {{ $t("settings.general.workspace.select-logo") }}
                </span>
                <p class="pl-1">
                  {{ $t("settings.general.workspace.drag-logo") }}
                </p>
              </div>
              <p class="text-xs text-gray-500 pointer-events-none">
                {{
                  $t("settings.general.workspace.logo-upload-tip", {
                    extension: supportImageExtensions.join(", "),
                    size: maxFileSizeInMiB,
                  })
                }}
              </p>
            </SingleFileSelector>
          </div>
        </div>
        <div class="flex">
          <button
            type="button"
            class="btn-primary ml-auto"
            :disabled="!allowSave"
            @click.prevent="uploadLogo"
          >
            <FeatureBadge
              feature="bb.feature.branding"
              class="text-white pointer-events-none"
            />
            {{ $t("common.update") }}
          </button>
        </div>
      </div>
    </div>
  </div>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.branding"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { isOwner } from "../utils";
import { Setting, brandingLogoSettingName } from "../types/setting";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

interface LocalState {
  displayName?: string;
  logoUrl?: string;
  logoFile: File | null;
  loading: boolean;
  showFeatureModal: boolean;
}

const maxFileSizeInMiB = 2;
const supportImageExtensions = [".jpg", ".jpeg", ".png", ".webp", ".svg"];

// convertFileToBase64 will convert a file into base64 string.
const convertFileToBase64 = (file: File) =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result);
    reader.onerror = (error) => reject(error);
  });

const store = useStore();
const { t } = useI18n();

const state = reactive<LocalState>({
  displayName: "",
  logoUrl: "",
  logoFile: null,
  loading: false,
  showFeatureModal: false,
});

store.dispatch("setting/fetchSetting").then(() => {
  const brandingLogoSetting: Setting = store.getters["setting/settingByName"](
    brandingLogoSettingName
  );
  state.logoUrl = brandingLogoSetting.value;
});

const currentUser = computed(() => store.getters["auth/currentUser"]());

const allowEdit = computed((): boolean => {
  return isOwner(currentUser.value.role);
});

const valid = computed((): boolean => {
  return !!state.displayName || !!state.logoFile;
});

const allowSave = computed((): boolean => {
  return (
    allowEdit.value && state.logoFile !== null && valid.value && !state.loading
  );
});

const hasBrandingFeature = computed((): boolean => {
  return store.getters["subscription/feature"]("bb.feature.branding");
});

const uploadLogo = async () => {
  if (!allowSave.value) {
    return;
  }
  if (!hasBrandingFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  if (!state.logoFile) {
    return;
  }

  state.loading = true;

  try {
    const fileInBase64 = await convertFileToBase64(state.logoFile);
    const setting: Setting = await store.dispatch(
      "setting/updateSettingByName",
      {
        name: brandingLogoSettingName,
        value: fileInBase64,
      }
    );

    state.logoFile = null;
    state.logoUrl = setting.value;

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("settings.general.workspace.logo-upload-succeed"),
    });
  } finally {
    state.loading = false;
  }
};

const onLogoSelect = (file: File) => {
  state.logoFile = file;
  state.logoUrl = URL.createObjectURL(file);
};
</script>
