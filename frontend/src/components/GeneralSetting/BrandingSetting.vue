<template>
  <div id="branding" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center gap-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
        <FeatureBadge :feature="PlanFeature.FEATURE_CUSTOM_LOGO" />
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
                class="flex flex-col gap-y-1 text-center justify-center items-center absolute top-0 bottom-0 left-0 right-0"
                :class="[state.logoUrl ? 'opacity-0 hover:opacity-90' : '']"
                :max-file-size-in-mi-b="maxFileSizeInMiB"
                :support-file-extensions="supportImageExtensions"
                :disabled="!allowEdit || !hasBrandingFeature"
                :show-no-data-placeholder="!state.logoUrl"
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
        <NButton
          v-if="allowEdit && !!state.logoUrl"
          secondary
          type="error"
          @click="() => (state.logoUrl = '')"
        >
          {{ $t("common.delete") }}
        </NButton>
      </div>
    </div>
  </div>

  <FeatureModal
    :feature="PlanFeature.FEATURE_CUSTOM_LOGO"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { featureToRef } from "@/store";
import { useActuatorV1Store } from "@/store/modules/v1/actuator";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";
import SingleFileSelector from "../SingleFileSelector.vue";

interface LocalState {
  logoUrl?: string;
  loading: boolean;
  showFeatureModal: boolean;
}

const props = defineProps<{
  title: string;
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

const state = reactive<LocalState>({
  logoUrl: settingV1Store.brandingLogo,
  loading: false,
  showFeatureModal: false,
});

const allowSave = computed((): boolean => {
  return state.logoUrl !== settingV1Store.brandingLogo;
});

const hasBrandingFeature = featureToRef(PlanFeature.FEATURE_CUSTOM_LOGO);

const doUpdate = async (content: string) => {
  if (state.loading) {
    return;
  }
  state.loading = true;
  try {
    await settingV1Store.updateWorkspaceProfile({
      payload: {
        brandingLogo: content,
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.branding_logo"],
      }),
    });
    useActuatorV1Store().setLogo(content);
  } finally {
    state.loading = false;
  }
};

const uploadLogo = async () => {
  await doUpdate(state.logoUrl ?? "");
};

const onLogoSelect = async (file: File) => {
  const fileInBase64 = await convertFileToBase64(file);
  state.logoUrl = fileInBase64;
};

defineExpose({
  title: props.title,
  isDirty: allowSave,
  update: uploadLogo,
  revert: () => (state.logoUrl = settingV1Store.brandingLogo),
});
</script>
