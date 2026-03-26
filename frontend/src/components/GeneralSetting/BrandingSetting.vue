<template>
  <div id="branding" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center gap-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
    </div>
    <div class="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
      <div>
        <label class="font-medium">{{ $t("settings.general.workspace.id") }}</label>
        <NInput :value="workspaceID" disabled class="mt-1" />
      </div>
      <div>
        <label class="font-medium">{{ $t("settings.general.workspace.title") }}</label>
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="['bb.workspaces.update']"
        >
          <NInput
            v-model:value="state.workspaceTitle"
            class="mt-1"
            :disabled="slotProps.disabled"
          />
        </PermissionGuardWrapper>
      </div>

      <div>
        <div class="mb-4 mt-4 lg:mt-0">
          <div class="flex items-center gap-x-2 font-medium">
            {{ $t("settings.general.workspace.logo") }}
            <FeatureBadge :feature="PlanFeature.FEATURE_CUSTOM_LOGO" />
          </div>
          <p class="mb-3 text-sm text-gray-400">
            {{ $t("settings.general.workspace.logo-aspect") }}
          </p>
          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="[
              'bb.workspaces.update'
            ]"
          >
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
                :disabled="slotProps.disabled || !hasBrandingFeature"
                :show-no-data-placeholder="!state.logoUrl"
                @on-select="onLogoSelect"
              />
            </div>
          </PermissionGuardWrapper>
        </div>
        <div v-if="!!state.logoUrl" class="flex justify-end gap-x-3">
          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="[
              'bb.settings.setWorkspaceProfile'
            ]"
          >
            <NButton
              secondary
              :disabled="slotProps.disabled"
              type="error"
              @click="() => (state.logoUrl = '')"
            >
              {{ $t("common.delete") }}
            </NButton>
          </PermissionGuardWrapper>
        </div>
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
import { NButton, NInput } from "naive-ui";
import { computed, reactive } from "vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { featureToRef, useWorkspaceV1Store } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";
import SingleFileSelector from "../SingleFileSelector.vue";

interface LocalState {
  logoUrl?: string;
  loading: boolean;
  workspaceTitle: string;
  showFeatureModal: boolean;
}

const props = defineProps<{
  title: string;
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

const workspaceStore = useWorkspaceV1Store();

const initialTitle = computed(() => {
  return workspaceStore.currentWorkspace?.title ?? "";
});

const state = reactive<LocalState>({
  logoUrl: workspaceStore.currentWorkspace?.logo,
  workspaceTitle: initialTitle.value,
  loading: false,
  showFeatureModal: false,
});

const workspaceID = computed(() => {
  const name = workspaceStore.currentWorkspace?.name ?? "";
  return name.replace(/^workspaces\//, "");
});

const allowSave = computed((): boolean => {
  return (
    state.workspaceTitle !== initialTitle.value ||
    state.logoUrl !== workspaceStore.currentWorkspace?.logo
  );
});

const hasBrandingFeature = featureToRef(PlanFeature.FEATURE_CUSTOM_LOGO);

const doUpdate = async (content: string) => {
  if (state.loading) {
    return;
  }
  state.loading = true;

  const updateMasks: string[] = [];
  const workspace = create(WorkspaceSchema, workspaceStore.currentWorkspace);
  try {
    if (
      state.workspaceTitle !== initialTitle.value &&
      state.workspaceTitle.trim() !== ""
    ) {
      workspace.title = state.workspaceTitle;
      updateMasks.push("title");
    }

    if (state.logoUrl !== workspaceStore.currentWorkspace?.logo) {
      workspace.logo = content;
      updateMasks.push("logo");
    }

    await workspaceStore.updateWorkspace(workspace, updateMasks);
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
  revert: () => {
    state.logoUrl = workspaceStore.currentWorkspace?.logo;
    state.workspaceTitle = initialTitle.value;
  },
});
</script>
