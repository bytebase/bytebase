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
                  ? $t("database.alter-schema").toLowerCase()
                  : $t("database.change-data").toLowerCase()
              }}
            </template>
            <template #action>
              <strong>
                {{
                  sqlEditorStore.mode === "BUNDLED"
                    ? isDDL
                      ? $t("database.alter-schema")
                      : $t("database.change-data")
                    : $t("sql-editor.admin-mode.self")
                }}
              </strong>
            </template>
            <template #reaction>
              {{
                sqlEditorStore.mode === "BUNDLED"
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
        v-if="sqlEditorStore.mode === 'BUNDLED'"
        class="flex justify-start items-center space-x-2"
      >
        <AdminModeButton @enter="$emit('close')" />
      </div>
      <div class="flex flex-1 justify-end items-center space-x-2">
        <NButton @click="handleClose">{{ $t("common.close") }}</NButton>
        <NButton
          v-if="sqlEditorStore.mode === 'BUNDLED'"
          type="primary"
          @click="gotoAlterSchema"
        >
          {{ isDDL ? $t("database.alter-schema") : $t("database.change-data") }}
        </NButton>
        <AdminModeButton v-else @enter="$emit('close')" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { parseSQL, isDDLStatement } from "@/components/MonacoEditor/sqlParser";
import {
  pushNotification,
  useDatabaseV1Store,
  useTabStore,
  useSQLEditorStore,
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
const sqlEditorStore = useSQLEditorStore();

const sqlStatement = computed(
  () => tabStore.currentTab.selectedStatement || tabStore.currentTab.statement
);

const isDDL = computed(() => {
  const { data } = parseSQL(sqlStatement.value);
  return data !== null ? isDDLStatement(data, "some") : false;
});

const handleClose = () => {
  emit("close");
};

const gotoAlterSchema = () => {
  const { databaseId } = tabStore.currentTab.connection;
  if (databaseId === String(UNKNOWN_ID)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-editor.goto-alter-schema-hint"),
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
      name: `[${database.name}] ${
        isDDL.value ? "Alter schema" : "Change Data"
      }`,
      project: database.projectEntity.uid,
      databaseList: databaseId,
      sql: sqlStatement.value,
    },
  });
};
</script>
