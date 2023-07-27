<template>
  <div class="w-[40vw] max-w-[calc(100vw-2rem)]">
    <NTabs v-model:value="view">
      <NTabPane name="my" :tab="$t('sheet.my-sheets')">
        <SheetTable view="my" @select-sheet="handleSelectSheet" />
      </NTabPane>
      <NTabPane name="shared" :tab="$t('sheet.shared-with-me')">
        <SheetTable view="shared" @select-sheet="handleSelectSheet" />
      </NTabPane>
      <NTabPane name="starred" :tab="$t('common.starred')">
        <SheetTable view="starred" @select-sheet="handleSelectSheet" />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from "naive-ui";
import { useI18n } from "vue-i18n";

import { Sheet } from "@/types/proto/v1/sheet_service";
import { emptyConnection, isSheetReadableV1 } from "@/utils";
import { UNKNOWN_ID } from "@/types";
import { pushNotification, useInstanceV1Store, useTabStore } from "@/store";
import { getInstanceAndDatabaseId } from "@/store/modules/v1/common";
import { useSheetPanelContext } from "./common";
import SheetTable from "./SheetTable";

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const tabStore = useTabStore();
const { view } = useSheetPanelContext();

const handleSelectSheet = async (sheet: Sheet) => {
  const openingSheetTab = tabStore.tabList.find(
    (tab) => tab.sheetName == sheet.name
  );

  if (!isSheetReadableV1(sheet)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    return false;
  }
  if (openingSheetTab) {
    // Switch to a sheet tab if it's open already.
    tabStore.setCurrentTabId(openingSheetTab.id);
  } else {
    // Open the sheet in a "temp" tab otherwise.
    tabStore.selectOrAddTempTab();
  }

  let insId = String(UNKNOWN_ID);
  let dbId = String(UNKNOWN_ID);
  if (sheet.database) {
    const [instanceName, databaseId] = getInstanceAndDatabaseId(sheet.database);
    const ins = await useInstanceV1Store().getOrFetchInstanceByName(
      `instances/${instanceName}`
    );
    insId = ins.uid;
    dbId = databaseId;
  }

  tabStore.updateCurrentTab({
    sheetName: sheet.name,
    name: sheet.title,
    statement: new TextDecoder().decode(sheet.content),
    isSaved: true,
    connection: {
      ...emptyConnection(),
      // TODO: legacy instance id.
      instanceId: insId,
      databaseId: dbId,
    },
  });

  emit("close");
};
</script>
