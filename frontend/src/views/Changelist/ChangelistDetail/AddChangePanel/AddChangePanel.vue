<template>
  <Drawer
    :show="showAddChangePanel"
    :close-on-esc="false"
    @close="showAddChangePanel = false"
  >
    <DrawerContent
      :title="$t('changelist.add-change.self')"
      class="w-[75vw] relative"
      style="max-width: calc(100vw - 8rem)"
    >
      <template #default>
        <div class="flex flex-col gap-y-4">
          <div class="flex items-center gap-x-8">
            <div class="textlabel">
              {{ $t("changelist.change-source.self") }}
            </div>
            <NRadioGroup v-model:value="changeSource">
              <NRadio value="CHANGE_HISTORY">
                {{ $t("common.change-history") }}
              </NRadio>
              <NRadio value="BRANCH">
                {{ $t("common.branch") }}
              </NRadio>
              <NRadio value="RAW_SQL">
                {{ $t("changelist.change-source.raw-sql") }}
              </NRadio>
            </NRadioGroup>
          </div>

          <ChangeHistoryForm v-if="changeSource === 'CHANGE_HISTORY'" />
          <BranchForm v-if="changeSource === 'BRANCH'" />
          <RawSQLForm v-if="changeSource === 'RAW_SQL'" />
        </div>

        <div
          v-if="isLoading"
          v-zindexable="{ enabled: true }"
          class="absolute bg-white/50 inset-0 flex flex-col items-center justify-center"
        >
          <BBSpin />
        </div>
      </template>

      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="showAddChangePanel = false">
            {{ $t("common.cancel") }}
          </NButton>
          <NTooltip :disabled="errors.length === 0">
            <template #trigger>
              <NButton
                type="primary"
                tag="div"
                :disabled="errors.length > 0"
                @click="doAddChange"
              >
                <span>{{ $t("common.add") }}</span>
                <span v-if="pendingAddChanges.length > 0" class="ml-1">
                  ({{ pendingAddChanges.length }})
                </span>
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="errors" />
            </template>
          </NTooltip>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { NRadio, NRadioGroup, NTooltip } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import ErrorList from "@/components/misc/ErrorList.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useChangelistDetailContext } from "../context";
import { provideAddChangeContext } from "./context";
import { BranchForm, ChangeHistoryForm, RawSQLForm } from "./form";

const { t } = useI18n();
const { showAddChangePanel } = useChangelistDetailContext();
const {
  changeSource,
  changesFromChangeHistory,
  changesFromBranch,
  changesFromRawSQL,
} = provideAddChangeContext();
const isLoading = ref(false);

const pendingAddChanges = computed(() => {
  switch (changeSource.value) {
    case "CHANGE_HISTORY":
      return changesFromChangeHistory.value;
    case "BRANCH":
      return changesFromBranch.value;
    case "RAW_SQL":
      return changesFromRawSQL.value;
  }
  console.warn("should never reach this line");
  return [];
});

const errors = asyncComputed(() => {
  const errors: string[] = [];

  if (pendingAddChanges.value.length === 0) {
    errors.push(t("changelist.add-change.select-at-least-one-change"));
  }

  return errors;
}, []);

const doAddChange = async () => {};

const reset = () => {
  changesFromChangeHistory.value = [];
  changesFromBranch.value = [];
  changesFromRawSQL.value = [];
};

watch(showAddChangePanel, (show) => {
  if (show) {
    reset();
  }
});
</script>
