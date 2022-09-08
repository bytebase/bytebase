<template>
  <div class="execute-hint w-160">
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
          <i18n-t keypath="sql-editor.want-to-change-schema">
            <template #changeschema>
              <NButton text :href="docLink" type="primary" target="_blank">
                {{ $t("sql-editor.change-schema") }}
              </NButton>
            </template>
          </i18n-t>
        </p>
        <p>
          <i18n-t keypath="sql-editor.go-to-alter-schema">
            <template #alterschema>
              <strong>
                {{
                  isDDLSQLStatement
                    ? $t("database.alter-schema")
                    : $t("database.change-data")
                }}
              </strong>
            </template>
          </i18n-t>
        </p>
      </section>
    </NAlert>

    <div class="execute-hint-content mt-4 flex justify-end space-x-2">
      <NButton @click="handleColse">{{ $t("common.close") }}</NButton>
      <NButton type="primary" @click="gotoAlterSchema">
        {{
          isDDLSQLStatement
            ? $t("database.alter-schema")
            : $t("database.change-data")
        }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { pushNotification, useTabStore, useDatabaseStore } from "@/store";
import {
  parseSQL,
  transformSQL,
  isDDLStatement,
} from "@/components/MonacoEditor/sqlParser";
import { UNKNOWN_ID } from "@/types";

const emit = defineEmits<{
  (e: "close"): void;
}>();

const DDLIssueTemplate = "bb.issue.database.schema.update";
const DMLIssueTemplate = "bb.issue.database.data.update";

const router = useRouter();
const { t } = useI18n();
const tabStore = useTabStore();

const sqlStatement = computed(
  () => tabStore.currentTab.selectedStatement || tabStore.currentTab.statement
);

const getParsedStatement = () => {
  const statement = sqlStatement.value;
  const { data } = parseSQL(statement);
  return data !== null ? transformSQL(data) : statement;
};

const isDDLSQLStatement = computed(() => {
  const statement = getParsedStatement();
  const { data } = parseSQL(statement);
  return data !== null ? isDDLStatement(data) : false;
});

const docLink =
  "https://bytebase.com/docs/concepts/schema-change-workflow#ui-workflow?source=console";

const handleColse = () => {
  emit("close");
};

const gotoAlterSchema = () => {
  const { databaseId } = tabStore.currentTab.connection;
  if (databaseId === UNKNOWN_ID) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-editor.goto-alter-schema-hint"),
    });
    return;
  }

  emit("close");

  const database = useDatabaseStore().getDatabaseById(databaseId);

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: isDDLSQLStatement.value ? DDLIssueTemplate : DMLIssueTemplate,
      name: `[${database.name}] ${
        isDDLSQLStatement.value ? "Alter schema" : "Change Data"
      }`,
      project: database.project.id,
      databaseList: databaseId,
      sql: getParsedStatement(),
    },
  });
};
</script>

<style scoped></style>
