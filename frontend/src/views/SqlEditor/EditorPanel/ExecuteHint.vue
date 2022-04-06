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
import {
  useNamespacedGetters,
  useNamespacedState,
} from "vuex-composition-helpers";

import { pushNotification, useTabStore } from "@/store";
import { UNKNOWN_ID } from "@/types";

import type { SqlEditorState, SqlEditorGetters } from "@/types";
import {
  parseSQL,
  transformSQL,
  isDDLStatement,
} from "@/components/MonacoEditor/sqlParser";
import {} from "@/store";

const emit = defineEmits<{
  (e: "close"): void;
}>();

const router = useRouter();
const { t } = useI18n();
const tabStore = useTabStore();

const { findProjectIdByDatabaseId } = useNamespacedGetters<SqlEditorGetters>(
  "sqlEditor",
  ["findProjectIdByDatabaseId"]
);
const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);

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

const ctx = connectionContext.value;

const docLink =
  "https://bytebase.com/docs/concepts/schema-change-workflow#ui-workflow";

const handleColse = () => {
  emit("close");
};

const gotoAlterSchema = () => {
  if (ctx.databaseId === UNKNOWN_ID) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-editor.goto-alter-schema-hint"),
    });
    return;
  }

  emit("close");

  const projectId = findProjectIdByDatabaseId.value(ctx.databaseId as number);
  const DDLIssueTemplate = "bb.issue.database.schema.update";
  const DMLIssueTemplate = "bb.issue.database.data.update";

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: isDDLSQLStatement.value ? DDLIssueTemplate : DMLIssueTemplate,
      name: `[${ctx.databaseName}] ${
        isDDLSQLStatement.value ? "Alter schema" : "Change Data"
      }`,
      project: projectId,
      databaseList: ctx.databaseId,
      sql: getParsedStatement(),
    },
  });
};
</script>

<style scoped></style>
