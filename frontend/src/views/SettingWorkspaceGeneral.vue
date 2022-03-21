<template>
  <div class="mt-2 space-y-6 divide-y divide-block-border">
    <div>
      <FeatureAttention
        v-if="!hasBrandingFeature"
        custom-class="mt-5"
        feature="bb.feature.branding"
        :description="$t('subscription.features.bb-feature-branding.desc')"
      />
      <div class="px-4 py-6 md:flex">
        <h1 class="text-left text-2xl font-bold w-1/3">
          {{ $t("settings.general.workspace.branding") }}
        </h1>
        <div class="flex-1 md:px-5">
          <!-- <div class="mb-5">
            <p class="mb-3">
              {{ $t("settings.general.workspace.display-name") }}
            </p>
            <input
              v-model="state.displayName"
              pattern="[a-z]+"
              type="text"
              name="org-display-name"
              class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md"
              placeholder="Enter a friendly name for your workspace"
            />
          </div> -->
          <div class="mb-5">
            <p class="">Logo</p>
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
                <div class="text-sm text-gray-600 inline-flex pointer-events-none">
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
                      size: maxFileSizeInMiB
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
              @click.prevent="doSave"
            >
              <FeatureBadge feature="bb.feature.branding" class="text-white" />
              {{ $t("common.update") }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive } from "vue";
import { useStore } from "vuex";
import { isOwner } from "../utils";

interface LocalState {
  displayName?: string;
  logoUrl?: string;
  logoFile: File | null;
  loading: boolean;
}

// getImageUrlFromBase64String will convert the image base64 string to the HTML image url.
const getImageUrlFromBase64String = (
  base64String: string | undefined
): string => {
  if (!base64String) return "";

  const decodedData = window.atob(base64String);
  const uInt8Array = new Uint8Array(decodedData.length);

  // Insert all character code into uInt8Array
  for (let i = 0; i < decodedData.length; ++i) {
    uInt8Array[i] = decodedData.charCodeAt(i);
  }

  const blob = new Blob([uInt8Array]);
  return URL.createObjectURL(blob);
};

export default defineComponent({
  name: "SettingWorkspaceGeneral",
  setup() {
    const store = useStore();

    const state = reactive<LocalState>({
      displayName: "",
      logoUrl: "",
      logoFile: null,
      loading: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const allowEdit = computed((): boolean => {
      return isOwner(currentUser.value.role);
    });

    const valid = computed((): boolean => {
      return !!state.displayName || !!state.logoFile
    });

    const changed = computed((): boolean => {
      return false;
    });

    const allowSave = computed((): boolean => {
      return changed.value && valid.value;
    });

    const hasBrandingFeature = computed((): boolean => {
      return store.getters["subscription/feature"]("bb.feature.branding");
    });

    const doSave = () => {
      // do nothing
    };

    const onLogoSelect = (file: File) => {
      state.logoFile = file;
      state.logoUrl = URL.createObjectURL(file);
    };

    return {
      state,
      allowEdit,
      allowSave,
      doSave,
      onLogoSelect,
      hasBrandingFeature,
      maxFileSizeInMiB: 2,
      supportImageExtensions: [
        ".jpg",
        ".jpeg",
        ".png",
        ".webp",
        ".svg",
      ],
    };
  },
  data() {
    return { placeholder: "{{ DB_NAME_PLACEHOLDER }}" };
  },
});
</script>
