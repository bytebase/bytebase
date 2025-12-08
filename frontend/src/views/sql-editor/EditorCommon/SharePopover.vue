<template>
  <div class="share-popover w-96 p-2 flex flex-col gap-y-4">
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
              class="border flex flex-row justify-start items-center px-2 py-1 rounded-sm"
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
        <div class="access-content flex flex-col gap-y-2 w-80">
          <div
            v-for="option in accessOptions"
            :key="option.label"
            class="p-2 rounded-xs flex justify-between"
            :class="[
              allowChangeAccess && 'cursor-pointer hover:bg-gray-200',
              option.value === currentAccess.value && 'bg-gray-200',
            ]"
            @click="handleChangeAccess(option)"
          >
            <div class="access-content--prefix">
              <div class="flex gap-x-2 items-center">
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
      <NInput :value="sharedTabLink" disabled />
      <div class="pl-2">
        <CopyButton
          quaternary
          :text="false"
          :size="'medium'"
          :content="sharedTabLink"
          :disabled="tabStore.currentTab?.status !== 'CLEAN'"
        />
      </div>
    </NInputGroup>
  </div>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { LockKeyholeIcon, UsersIcon } from "lucide-vue-next";
import { NInput, NInputGroup, NInputGroupLabel, NPopover } from "naive-ui";
import { computed, h, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { CopyButton } from "@/components/v2";
import { SQL_EDITOR_WORKSHEET_MODULE } from "@/router/sqlEditor";
import {
  pushNotification,
  useCurrentUserV1,
  useSettingV1Store,
  useSQLEditorTabStore,
  useWorkSheetStore,
} from "@/store";
import type { AccessOption } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { extractProjectResourceName, extractWorksheetUID } from "@/utils";

const props = defineProps<{
  worksheet?: Worksheet;
}>();

const emit = defineEmits<{
  (event: "on-updated"): void;
}>();

const { t } = useI18n();

const router = useRouter();
const tabStore = useSQLEditorTabStore();
const worksheetV1Store = useWorkSheetStore();
const settingStore = useSettingV1Store();
const me = useCurrentUserV1();

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});

const workspaceExternalURL = computed(
  () => settingStore.workspaceProfileSetting?.externalUrl
);

const accessOptions = computed<AccessOption[]>(() => {
  return [
    {
      label: t("sql-editor.private"),
      value: Worksheet_Visibility.PRIVATE,
      description: t("sql-editor.private-desc"),
      icon: h(LockKeyholeIcon),
    },
    {
      label: t("sql-editor.project-read"),
      value: Worksheet_Visibility.PROJECT_READ,
      description: t("sql-editor.project-read-desc"),
      icon: h(UsersIcon),
    },
    {
      label: t("sql-editor.project-write"),
      value: Worksheet_Visibility.PROJECT_WRITE,
      description: t("sql-editor.project-write-desc"),
      icon: h(UsersIcon),
    },
  ];
});

const allowChangeAccess = computed(() => {
  return props.worksheet?.creator === `users/${me.value.email}`;
});

const currentAccess = ref<AccessOption>(accessOptions.value[0]);
const isShowAccessPopover = ref(false);

const handleChangeAccess = async (option: AccessOption) => {
  // only creator can change access
  if (allowChangeAccess.value && props.worksheet) {
    currentAccess.value = option;
    await worksheetV1Store.patchWorksheet(
      {
        ...props.worksheet,
        visibility: currentAccess.value.value,
      },
      ["visibility"]
    );

    if (sharedTabLink.value && isSupported.value) {
      await copyTextToClipboard(sharedTabLink.value);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } else {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    }

    emit("on-updated");
  }
  isShowAccessPopover.value = false;
};

const sharedTabLink = computed(() => {
  if (!props.worksheet) {
    return "";
  }

  const route = router.resolve({
    name: SQL_EDITOR_WORKSHEET_MODULE,
    params: {
      project: extractProjectResourceName(props.worksheet.project),
      sheet: extractWorksheetUID(props.worksheet.name),
    },
  });
  return new URL(
    route.href,
    workspaceExternalURL.value || window.location.origin
  ).href;
});

onMounted(() => {
  if (props.worksheet) {
    const { visibility } = props.worksheet;
    const idx = accessOptions.value.findIndex(
      (item) => item.value === visibility
    );
    currentAccess.value =
      idx !== -1 ? accessOptions.value[idx] : accessOptions.value[0];
  }
});
</script>
