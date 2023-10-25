<template>
  <Drawer
    :show="showAddChangePanel"
    :close-on-esc="false"
    @close="showAddChangePanel = false"
  >
    <DrawerContent
      :title="$t('changelist.add-change.self')"
      class="w-[80vw] relative"
      style="max-width: calc(100vw - 8rem)"
    >
      <template #default>
        <div class="flex flex-col gap-y-4 min-h-full">
          <div class="flex items-center gap-x-8">
            <div class="textlabel">
              {{ $t("changelist.change-source.self") }}
            </div>
            <NRadioGroup v-model:value="changeSource">
              <NRadio value="CHANGE_HISTORY">
                <div class="flex items-center">
                  <History :size="16" class="mr-1" />
                  {{ $t("common.change-history") }}
                </div>
              </NRadio>
              <NRadio value="BRANCH">
                <div class="flex items-center">
                  <GitBranch :size="16" class="mr-1" />{{ $t("common.branch") }}
                </div>
              </NRadio>
              <NRadio value="RAW_SQL">
                <div class="flex items-center">
                  <File :size="16" class="mr-1" />
                  {{ $t("changelist.change-source.raw-sql") }}
                </div>
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

          <ErrorTipsButton
            :errors="errors"
            :button-props="{
              type: 'primary',
            }"
            @click="doAddChange"
          >
            <span>{{ $t("common.add") }}</span>
            <span
              v-if="changeSource !== 'RAW_SQL' && pendingAddChanges.length > 0"
              class="ml-1"
            >
              ({{ pendingAddChanges.length }})
            </span>
          </ErrorTipsButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { File, GitBranch, History } from "lucide-vue-next";
import { NRadio, NRadioGroup } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { fetchBaselineMetadataOfBranch } from "@/components/SchemaEditorV1/utils/branch";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import { schemaDesignServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useChangeHistoryStore,
  useChangelistStore,
  useLocalSheetStore,
} from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import {
  Changelist_Change as Change,
  Changelist,
} from "@/types/proto/v1/changelist_service";
import {
  getChangelistChangeSourceType,
  getSheetStatement,
  isLocalSheet,
  setSheetStatement,
} from "@/utils";
import { useChangelistDetailContext } from "../context";
import { provideAddChangeContext } from "./context";
import { BranchForm, ChangeHistoryForm, RawSQLForm } from "./form";
import { emptyRawSQLChange } from "./utils";

const { t } = useI18n();
const { project, changelist, showAddChangePanel } =
  useChangelistDetailContext();
const {
  changeSource,
  changesFromChangeHistory,
  changesFromBranch,
  changeFromRawSQL,
} = provideAddChangeContext();
const isLoading = ref(false);

const pendingAddChanges = computed(() => {
  switch (changeSource.value) {
    case "CHANGE_HISTORY":
      return changesFromChangeHistory.value;
    case "BRANCH":
      return changesFromBranch.value;
    case "RAW_SQL":
      return [changeFromRawSQL.value];
  }
  console.warn("should never reach this line");
  return [];
});

const errors = asyncComputed(() => {
  const errors: string[] = [];

  if (pendingAddChanges.value.length === 0) {
    errors.push(t("changelist.error.select-at-least-one-change"));
  }
  if (changeSource.value === "RAW_SQL") {
    const name = changeFromRawSQL.value.sheet;
    const sheet = useLocalSheetStore().getOrCreateSheetByName(name);
    const statement = getSheetStatement(sheet);
    if (statement.trim().length === 0) {
      errors.push(t("changelist.error.sql-cannot-be-empty"));
    }
  }

  return errors;
}, []);

const doAddChange = async () => {
  if (errors.value.length > 0) return;

  isLoading.value = true;
  const createSheetForPendingAddChange = async (pendingAddChange: Change) => {
    const change = { ...pendingAddChange };
    if (isLocalSheet(change.sheet)) {
      const localSheetStore = useLocalSheetStore();
      const sheet = localSheetStore.getOrCreateSheetByName(change.sheet);
      const sourceType = getChangelistChangeSourceType(change);
      if (sourceType === "CHANGE_HISTORY") {
        const { statement } =
          await useChangeHistoryStore().exportChangeHistoryFullStatementByName(
            change.source
          );
        setSheetStatement(sheet, statement);
      }
      if (sourceType === "BRANCH") {
        // For branch changes, use its diff DDL
        const branch = await useSchemaDesignStore().fetchSchemaDesignByName(
          change.source,
          false /* !useCache */
        );
        const source = await fetchBaselineMetadataOfBranch(branch);
        const target = branch.schemaMetadata;

        const { diff } = await schemaDesignServiceClient.diffMetadata({
          sourceMetadata: source,
          targetMetadata: target,
          engine: branch.engine,
        });
        setSheetStatement(sheet, diff);
      }
      const created = await localSheetStore.saveLocalSheetToRemote(sheet);
      change.sheet = created.name;
    }

    return change;
  };
  try {
    const newChanges: Change[] = [];
    for (let i = 0; i < pendingAddChanges.value.length; i++) {
      newChanges.push(
        await createSheetForPendingAddChange(pendingAddChanges.value[i])
      );
    }
    const changelistPatch = {
      ...Changelist.fromPartial(changelist.value),
      changes: [...changelist.value.changes, ...newChanges],
    };
    await useChangelistStore().patchChangelist(changelistPatch, ["changes"]);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.added"),
    });
    showAddChangePanel.value = false;
  } finally {
    isLoading.value = false;
  }
};

const reset = () => {
  changeSource.value = "CHANGE_HISTORY";
  changesFromChangeHistory.value = [];
  changesFromBranch.value = [];
  changeFromRawSQL.value = emptyRawSQLChange(project.value.name);
};

watch(showAddChangePanel, (show) => {
  if (show) {
    reset();
  }
});
</script>
