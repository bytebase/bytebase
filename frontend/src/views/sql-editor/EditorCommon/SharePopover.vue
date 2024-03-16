<template>
  <div class="share-popover w-96 p-2 space-y-4">
    <section class="w-full flex flex-row justify-between items-center">
      <div class="pr-4">
        <h2 class="text-lg font-semibold">{{ $t("common.share") }}</h2>
      </div>
      <NPopover
        trigger="click"
        :show="isShowAccessPopover"
        :disabled="!allowChangeAccess"
      >
        <template #trigger>
          <div
            class="flex items-center"
            :class="allowChangeAccess ? 'cursor-pointer' : 'cursor-not-allowed'"
            @click="isShowAccessPopover = !isShowAccessPopover"
          >
            <span class="pr-2">{{ $t("sql-editor.link-access") }}:</span>
            <div
              class="border flex flex-row justify-start items-center px-2 py-1 rounded"
              :class="
                allowChangeAccess
                  ? 'hover:border-accent'
                  : 'border-gray-200 text-gray-400'
              "
            >
              <strong>{{ currentAccess.label }}</strong>
              <heroicons-solid:chevron-down />
            </div>
          </div>
        </template>
        <div class="access-content space-y-2 w-80">
          <div
            v-for="option in accessOptions"
            :key="option.label"
            class="p-2 rounded-sm flex justify-between"
            :class="[
              allowChangeAccess && 'cursor-pointer hover:bg-gray-200',
              option.value === currentAccess.value && 'bg-gray-200',
            ]"
            @click="handleChangeAccess(option)"
          >
            <div class="access-content--prefix">
              <div class="flex space-x-2 items-center">
                <component :is="option.icon" class="h-5 w-5" />
                <h2 class="text-md flex">
                  {{ option.label }}
                </h2>
              </div>
              <span class="text-xs textinfolabel">
                {{ option.description }}
              </span>
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
      <NInputGroupLabel>
        <div
          class="w-full h-full flex flex-row items-center justify-center m-auto"
        >
          <heroicons-solid:link class="w-5 h-auto" />
        </div>
      </NInputGroupLabel>
      <NInput v-model:value="sharedTabLink" disabled />
      <NButton
        class="w-20"
        :type="copied ? 'success' : 'primary'"
        :disabled="tabStore.currentTab?.status !== 'CLEAN'"
        @click="handleCopy"
      >
        <heroicons-solid:check v-if="copied" class="h-4 w-4" />
        {{ copied ? $t("common.copied") : $t("common.copy") }}
      </NButton>
    </NInputGroup>
  </div>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { LockKeyholeIcon, UsersIcon } from "lucide-vue-next";
import {
  NButton,
  NInput,
  NInputGroup,
  NInputGroupLabel,
  NPopover,
} from "naive-ui";
import { ref, computed, onMounted, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { SQL_EDITOR_SHARE_MODULE } from "@/router/sqlEditor";
import {
  pushNotification,
  useSQLEditorTabStore,
  useWorkSheetStore,
  useWorkSheetAndTabStore,
} from "@/store";
import { AccessOption } from "@/types";
import { Worksheet_Visibility } from "@/types/proto/v1/worksheet_service";
import { worksheetSlugV1 } from "@/utils";

const { t } = useI18n();

const router = useRouter();
const tabStore = useSQLEditorTabStore();
const worksheetV1Store = useWorkSheetStore();
const sheetAndTabStore = useWorkSheetAndTabStore();

const accessOptions = computed<AccessOption[]>(() => {
  return [
    {
      label: t("sql-editor.private"),
      value: Worksheet_Visibility.VISIBILITY_PRIVATE,
      description: t("sql-editor.private-desc"),
      icon: h(LockKeyholeIcon),
    },
    {
      label: t("sql-editor.project-read"),
      value: Worksheet_Visibility.VISIBILITY_PROJECT_READ,
      description: t("sql-editor.project-read-desc"),
      icon: h(UsersIcon),
    },
    {
      label: t("sql-editor.project-write"),
      value: Worksheet_Visibility.VISIBILITY_PROJECT_WRITE,
      description: t("sql-editor.project-write-desc"),
      icon: h(UsersIcon),
    },
  ];
});

const sheet = computed(() => {
  return sheetAndTabStore.currentSheet;
});
const allowChangeAccess = computed(() => {
  return sheetAndTabStore.isCreator;
});

const currentAccess = ref<AccessOption>(accessOptions.value[0]);
const isShowAccessPopover = ref(false);

const updateWorksheet = () => {
  if (sheet.value) {
    worksheetV1Store.patchSheet(
      {
        name: sheet.value.name,
        visibility: currentAccess.value.value,
      },
      ["visibility"]
    );
  }
};

const handleChangeAccess = (option: AccessOption) => {
  // only creator can change access
  if (allowChangeAccess.value) {
    currentAccess.value = option;
    updateWorksheet();
  }
  isShowAccessPopover.value = false;
};

const sharedTabLink = computed(() => {
  if (!sheet.value) {
    return "";
  }

  const route = router.resolve({
    name: SQL_EDITOR_SHARE_MODULE,
    params: {
      sheetSlug: worksheetSlugV1(sheet.value),
    },
  });
  return new URL(route.href, window.location.origin).href;
});

const { copy, copied } = useClipboard({
  source: sharedTabLink.value,
  legacy: true,
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
