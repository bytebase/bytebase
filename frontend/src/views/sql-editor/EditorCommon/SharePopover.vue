<template>
  <div class="share-popover w-96 p-2 space-y-4">
    <section class="w-full flex flex-row justify-between items-center">
      <div class="pr-4">
        <h2 class="text-lg font-semibold">{{ $t("common.share") }}</h2>
      </div>
      <NPopover trigger="click" :show="isShowAccessPopover">
        <template #trigger>
          <div
            class="flex items-center cursor-pointer"
            @click="isShowAccessPopover = !isShowAccessPopover"
          >
            <span class="pr-2">{{ $t("sql-editor.link-access") }}:</span>
            <div
              class="border flex flex-row justify-start items-center px-2 py-1 rounded hover:border-accent"
            >
              <strong>{{ currentAccess.label }}</strong>
              <heroicons-solid:chevron-down />
            </div>
          </div>
        </template>
        <div class="access-content space-y-2 w-80">
          <div
            v-for="(option, idx) in accessOptions"
            :key="option.label"
            class="p-2 rounded-sm flex justify-between"
            :class="[
              isCreator && 'cursor-pointer hover:bg-gray-200',
              option.value === currentAccess.value && 'bg-gray-200',
            ]"
            @click="handleChangeAccess(option)"
          >
            <div class="access-content--prefix flex">
              <div v-if="idx === 0" class="mt-1">
                <heroicons-outline:lock-closed class="h-5 w-5" />
              </div>
              <div v-if="idx === 1" class="mt-1">
                <heroicons-outline:user-group class="h-5 w-5" />
              </div>
              <div v-if="idx === 2" class="mt-1">
                <heroicons-outline:globe class="h-5 w-5" />
              </div>
              <section class="flex flex-col pl-2">
                <h2 class="text-md flex">
                  {{ option.label }}
                </h2>
                <h3 class="text-xs">
                  {{ option.description }}
                </h3>
              </section>
            </div>
            <div
              v-show="option.value === currentAccess.value"
              class="access-content--suffix flex items-center"
            >
              <heroicons-solid:check class="h-5 w-5" />
            </div>
          </div>
        </div>
      </NPopover>
    </section>
    <NInputGroup class="flex items-center justify-center">
      <n-input-group-label>
        <div
          class="w-full h-full flex flex-row items-center justify-center m-auto"
        >
          <heroicons-solid:link class="w-5 h-auto" />
        </div>
      </n-input-group-label>
      <n-input v-model:value="sharedTabLink" disabled />
      <NButton
        class="w-20"
        :type="copied ? 'success' : 'primary'"
        :disabled="!tabStore.currentTab.isSaved"
        @click="handleCopy"
      >
        <heroicons-solid:check v-if="copied" class="h-4 w-4" />
        {{ copied ? $t("common.copied") : $t("common.copy") }}
      </NButton>
    </NInputGroup>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from "vue";
import { useClipboard } from "@vueuse/core";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useTabStore,
  useSheetV1Store,
  useSheetAndTabStore,
} from "@/store";
import { AccessOption } from "@/types";
import { sheetSlugV1 } from "@/utils";
import { Sheet_Visibility } from "@/types/proto/v1/sheet_service";

const { t } = useI18n();

const tabStore = useTabStore();
const sheetV1Store = useSheetV1Store();
const sheetAndTabStore = useSheetAndTabStore();

const accessOptions = computed<AccessOption[]>(() => {
  return [
    {
      label: t("sql-editor.private"),
      value: Sheet_Visibility.VISIBILITY_PRIVATE,
      description: t("sql-editor.private-desc"),
    },
    {
      label: t("common.project"),
      value: Sheet_Visibility.VISIBILITY_PROJECT,
      description: t("sql-editor.project-desc"),
    },
    {
      label: t("sql-editor.public"),
      value: Sheet_Visibility.VISIBILITY_PUBLIC,
      description: t("sql-editor.public-desc"),
    },
  ];
});

const sheet = computed(() => {
  return sheetAndTabStore.currentSheet;
});
const isCreator = computed(() => {
  return sheetAndTabStore.isCreator;
});

const currentAccess = ref<AccessOption>(accessOptions.value[0]);
const isShowAccessPopover = ref(false);

const updateSheet = () => {
  if (sheet.value) {
    sheetV1Store.patchSheet({
      name: sheet.value.name,
      visibility: currentAccess.value.value,
    });
  }
};

const handleChangeAccess = (option: AccessOption) => {
  // only creator can change access
  if (isCreator.value) {
    currentAccess.value = option;
    updateSheet();
  }
  isShowAccessPopover.value = false;
};

const sharedTabLink = computed(() => {
  if (!sheet.value) {
    return "";
  }
  return `${window.location.origin}/sql-editor/sheet/${sheetSlugV1(
    sheet.value
  )}`;
});

const { copy, copied } = useClipboard({
  source: sharedTabLink.value,
});

const handleCopy = async () => {
  await copy();
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-editor.notify.copy-share-link"),
  });
};

onMounted(() => {
  if (sheet.value) {
    const { visibility } = sheet.value;
    const idx = accessOptions.value.findIndex(
      (item) => item.value === visibility
    );
    currentAccess.value =
      idx !== -1 ? accessOptions.value[idx] : accessOptions.value[0];
  }
});
</script>
