<template>
  <div class="share-popover w-112 p-2 space-y-4">
    <section class="flex justify-between">
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
            <strong>{{ currentAccess.label }}</strong>
            <heroicons-solid:chevron-down />
          </div>
        </template>
        <div class="access-content space-y-2 w-80">
          <div
            v-for="(option, idx) in accessOptions"
            :key="option.label"
            class="p-2 rounded-sm flex justify-between"
            :class="[
              creatorAccessStyle,
              {
                'bg-accent text-white': option.value === currentAccess.value,
              },
            ]"
            @click="handleChangeAccess(option)"
          >
            <div class="access-content--prefix flex">
              <div v-if="idx === 0" class="mt-1">
                <heroicons-outline:user class="h-5 w-5" />
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
                <h3 class="text-xs" :class="creatorAccessDescStyle">
                  {{ option.desc }}
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
      <n-input-group-label class="flex items-center">
        <heroicons-solid:link class="h-4 w-4" />
      </n-input-group-label>
      <n-input v-model:value="sharedTabLink" disabled />
      <NButton
        class="w-20"
        :type="copied ? 'success' : 'primary'"
        @click="handleCopy"
      >
        <heroicons-solid:check v-if="copied" class="h-4 w-4" />
        {{ copied ? $t("common.copied") : $t("common.copy") }}
      </NButton>
    </NInputGroup>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, defineEmits } from "vue";
import { useClipboard } from "@vueuse/core";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";
import {
  useNamespacedGetters,
  useNamespacedState,
  useNamespacedActions,
} from "vuex-composition-helpers";
import slug from "slug";

import {
  TabGetters,
  SqlEditorState,
  SheetGetters,
  SheetActions,
  AccessOption,
} from "../../../types";

const emit = defineEmits<{
  (e: "close"): void;
}>();

const store = useStore();
const { t } = useI18n();

const accessOptions = computed<AccessOption[]>(() => {
  return [
    {
      label: t("sql-editor.private"),
      value: "PRIVATE",
      desc: t("sql-editor.private-desc"),
    },
    {
      label: t("common.project"),
      value: "PROJECT",
      desc: t("sql-editor.project-desc"),
    },
    {
      label: t("sql-editor.public"),
      value: "PUBLIC",
      desc: t("sql-editor.public-desc"),
    },
  ];
});

const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);
const { currentTab } = useNamespacedGetters<TabGetters>("tab", ["currentTab"]);
const { currentSheet, isCreator } = useNamespacedGetters<SheetGetters>(
  "sheet",
  ["currentSheet", "isCreator"]
);
const { patchSheetById } = useNamespacedActions<SheetActions>("sheet", [
  "patchSheetById",
]);

const ctx = connectionContext.value;

const host = window.location.host;
const connectionSlug = [
  slug(ctx.instanceName as string),
  ctx.instanceId,
  slug(ctx.databaseName as string),
  ctx.databaseId,
].join("_");

const currentAccess = ref<AccessOption>(accessOptions.value[0]);
const isShowAccessPopover = ref(false);

const creatorAccessStyle = computed(() => {
  return isCreator.value
    ? "cursor-pointer hover:bg-accent hover:text-white"
    : "bg-transparent text-gray-300 cursor-not-allowed";
});
const creatorAccessDescStyle = computed(() => {
  return isCreator.value ? "text-gray-400" : "text-gray-300 cursor-not-allowed";
});

const updateSheet = () => {
  if (currentTab.value.sheetId) {
    patchSheetById({
      id: currentTab.value.sheetId,
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

const sheetSlug = `${slug(currentTab.value.label)}_${currentTab.value.sheetId}`;
const sharedTabLink = ref(`${host}/sql-editor/${connectionSlug}/${sheetSlug}`);

const { copy, copied } = useClipboard({
  source: sharedTabLink.value,
});

const handleCopy = async () => {
  await copy();
  store.dispatch("notification/pushNotification", {
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-editor.copy-share-link"),
  });
  emit("close");
};

onMounted(() => {
  if (currentSheet.value) {
    const { visibility } = currentSheet.value;
    const idx = accessOptions.value.findIndex(
      (item) => item.value === visibility
    );
    currentAccess.value =
      idx !== -1 ? accessOptions.value[idx] : accessOptions.value[0];
  }
});
</script>
