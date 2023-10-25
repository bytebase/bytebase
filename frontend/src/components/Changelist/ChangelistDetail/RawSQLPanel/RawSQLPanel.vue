<template>
  <Drawer :show="show" :close-on-esc="!dirty" @close="$emit('close')">
    <DrawerContent
      :title="title"
      class="relative"
      style="width: 75vw; max-width: calc(100vw - 8rem)"
    >
      <template #default>
        <div class="h-full flex flex-col gap-y-4 flex-1 overflow-hidden">
          <BBAttention
            v-if="isSheetOversize"
            :style="`WARN`"
            :title="$t('issue.statement-from-sheet-warning')"
          >
            <template #action>
              <DownloadSheetButton
                v-if="sheetName"
                :sheet="sheetName"
                size="small"
              />
            </template>
          </BBAttention>
          <RawSQLEditor
            v-if="sheet"
            v-model:statement="statement"
            :readonly="!allowEdit"
            :is-sheet-oversize="isSheetOversize"
            class="flex-1 overflow-hidden relative"
            @upload="handleUploadEvent"
          />
        </div>
        <div
          v-if="isUpdating"
          v-zindexable="{ enabled: true }"
          class="absolute bg-white/50 inset-0 flex flex-col items-center justify-center"
        >
          <BBSpin />
        </div>
      </template>

      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>

          <ErrorTipsButton
            v-if="allowEdit"
            :errors="errors"
            :button-props="{
              type: 'primary',
            }"
            @click="doSaveChange"
          >
            {{ $t("common.save") }}
          </ErrorTipsButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { zindexable as vZindexable } from "vdirs";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention } from "@/bbkit";
import DownloadSheetButton from "@/components/Sheet/DownloadSheetButton.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { pushNotification, useSheetV1Store } from "@/store";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { getSheetStatement, readFileAsync, setSheetStatement } from "@/utils";
import RawSQLEditor from "../RawSQLEditor";
import { useChangelistDetailContext } from "../context";

const props = defineProps<{
  sheetName?: string;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const sheetStore = useSheetV1Store();
const { t } = useI18n();
const { allowEdit } = useChangelistDetailContext();
const title = ref(t("changelist.change-source.raw-sql"));
const statement = ref("");
const isUpdating = ref(false);

const sheet = computed(() => {
  const { sheetName } = props;
  if (!sheetName) return undefined;
  return sheetStore.getSheetByName(sheetName);
});

const show = computed(() => {
  return sheet.value !== undefined;
});

const isSheetOversize = computed(() => {
  if (!sheet.value) return false;
  return getSheetStatement(sheet.value).length < sheet.value.contentSize;
});

const dirty = computed(() => {
  if (!sheet.value) return false;
  return statement.value !== getSheetStatement(sheet.value);
});

const errors = computed(() => {
  const errors: string[] = [];
  if (statement.value.trim().length === 0) {
    errors.push(t("changelist.error.sql-cannot-be-empty"));
  }
  if (!dirty.value) {
    errors.push(t("changelist.error.nothing-changed"));
  }
  return errors;
});

const reset = () => {
  if (!sheet.value) {
    statement.value = "";
  } else {
    statement.value = getSheetStatement(sheet.value);
  }
};

const handleUploadEvent = async (e: Event) => {
  try {
    const { content } = await readFileAsync(e, 100);
    statement.value = content;
  } finally {
    // Nothing
  }
};

const doSaveChange = async () => {
  if (!sheet.value) {
    return;
  }
  if (errors.value.length > 0) {
    return;
  }

  try {
    isUpdating.value = true;
    const patch = Sheet.fromPartial({
      name: sheet.value.name,
    });
    setSheetStatement(patch, statement.value);
    await sheetStore.patchSheet(patch, ["content"]);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    emit("close");
  } finally {
    isUpdating.value = false;
  }
};

watch(sheet, reset, { immediate: true });

watch(
  show,
  (show) => {
    if (show) {
      reset();
    }
  },
  { immediate: true }
);
</script>
