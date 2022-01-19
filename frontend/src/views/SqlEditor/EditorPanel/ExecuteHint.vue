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
                {{
                $t("sql-editor.change-schema")
                }}
              </NButton>
            </template>
          </i18n-t>
        </p>
        <p>
          <i18n-t keypath="sql-editor.go-to-alter-schema">
            <template #alterschema>
              <strong>{{ $t("database.alter-schema") }}</strong>
            </template>
          </i18n-t>
        </p>
      </section>
    </NAlert>

    <div class="execute-hint-content mt-4 flex justify-end space-x-2">
      <NButton @click="handleColse">{{ $t("common.close") }}</NButton>
      <NButton type="primary" @click="gotoAlterSchema">
        {{
        $t("database.alter-schema")
        }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, defineEmits } from "vue";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";

import {
  useNamespacedGetters,
  useNamespacedState,
} from "vuex-composition-helpers";

import type {
  SqlEditorState,
  SqlEditorGetters,
  EditorSelectorGetters,
} from "../../../types";
import {
  parseSQL,
  transformSQL,
} from "../../../components/MonacoEditor/sqlParser";

const emit = defineEmits<{
  (e: "close"): void;
}>();

const router = useRouter();
const store = useStore();
const { t } = useI18n();

const { findProjectIdByDatabaseId } = useNamespacedGetters<SqlEditorGetters>(
  "sqlEditor",
  ["findProjectIdByDatabaseId"]
);
const { connectionContext } = useNamespacedState<SqlEditorState>("sqlEditor", [
  "connectionContext",
]);
const { currentTab } = useNamespacedGetters<EditorSelectorGetters>(
  "editorSelector",
  ["currentTab"]
);

const parsedStatement = computed(() => {
  const sqlStatement =
    currentTab.value.selectedStatement || currentTab.value.queryStatement;
  const { data } = parseSQL(sqlStatement);
  return data !== null ? transformSQL(data) : sqlStatement;
});

const ctx = connectionContext.value;

const docLink =
  "https://docs.bytebase.com/concepts/schema-change-workflow#ui-workflow";

const handleColse = () => {
  emit("close");
};

const gotoAlterSchema = () => {
  if (ctx.databaseId === 0) {
    store.dispatch("notification/pushNotification", {
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-editor.goto-alter-schema-hint"),
    });
    return;
  }

  emit("close");

  const projectId = findProjectIdByDatabaseId.value(ctx.databaseId as number);

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.database.schema.update",
      name: `[${ctx.databaseName}] Alter schema`,
      project: projectId,
      databaseList: ctx.databaseId,
      sql: parsedStatement.value,
    },
  });
};
</script>

<style scoped></style>
