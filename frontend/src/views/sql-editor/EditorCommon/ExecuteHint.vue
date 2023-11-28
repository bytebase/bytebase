<template>
  <div class="execute-hint w-112">
    <NAlert type="info">
      <section class="space-y-2">
        <p>
          <i18n-t keypath="sql-editor.only-select-allowed">
            <template #select>
              <strong>SELECT</strong>
            </template>
          </i18n-t>
        </p>
        <p>
          <i18n-t keypath="sql-editor.want-to-action">
            <template #want>
              {{
                isDDL
                  ? $t("database.edit-schema").toLowerCase()
                  : $t("database.change-data").toLowerCase()
              }}
            </template>
            <template #action>
              <strong>
                {{
                  showActionButtons
                    ? isDDL
                      ? $t("database.edit-schema")
                      : $t("database.change-data")
                    : $t("sql-editor.admin-mode.self")
                }}
              </strong>
            </template>
            <template #reaction>
              {{
                showActionButtons
                  ? $t("sql-editor.and-submit-an-issue")
                  : $t("sql-editor.to-enable-admin-mode")
              }}
            </template>
          </i18n-t>
        </p>
      </section>
    </NAlert>

    <div class="execute-hint-content mt-4 flex justify-between">
      <div
        v-if="showActionButtons"
        class="flex justify-start items-center space-x-2"
      >
        <AdminModeButton @enter="$emit('close')" />
      </div>
      <div class="flex flex-1 justify-end items-center space-x-2">
        <NButton @click="handleClose">{{ $t("common.close") }}</NButton>
        <NButton
          v-if="showActionButtons"
          type="primary"
          @click="gotoCreateIssue"
        >
          {{ isDDL ? $t("database.edit-schema") : $t("database.change-data") }}
        </NButton>
        <AdminModeButton v-else @enter="$emit('close')" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { parseSQL, isDDLStatement } from "@/components/MonacoEditor/sqlParser";
import {
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useTabStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import AdminModeButton from "./AdminModeButton.vue";

const emit = defineEmits<{
  (e: "close"): void;
}>();

const DDLIssueTemplate = "bb.issue.database.schema.update";
const DMLIssueTemplate = "bb.issue.database.data.update";

const router = useRouter();
const { t } = useI18n();
const tabStore = useTabStore();
const { pageMode } = storeToRefs(useActuatorV1Store());

const sqlStatement = computed(
  () => tabStore.currentTab.selectedStatement || tabStore.currentTab.statement
);

const isDDL = computedAsync(async () => {
  const { data } = await parseSQL(sqlStatement.value);
  return data !== null ? isDDLStatement(data, "some") : false;
}, false);

const showActionButtons = computed(() => pageMode.value === "BUNDLED");

const handleClose = () => {
  emit("close");
};

const gotoCreateIssue = () => {
  const { databaseId } = tabStore.currentTab.connection;
  if (databaseId === String(UNKNOWN_ID)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-editor.goto-edit-schema-hint"),
    });
    return;
  }

  emit("close");

  const database = useDatabaseV1Store().getDatabaseByUID(databaseId);

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: isDDL.value ? DDLIssueTemplate : DMLIssueTemplate,
      name: `[${database.databaseName}] ${
        isDDL.value ? "Alter schema" : "Change Data"
      }`,
      project: database.projectEntity.uid,
      databaseList: databaseId,
      sql: sqlStatement.value,
    },
  });
};
</script>
