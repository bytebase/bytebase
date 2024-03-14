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
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { extractProjectResourceName } from "@/utils";
import AdminModeButton from "./AdminModeButton.vue";

const emit = defineEmits<{
  (e: "close"): void;
}>();

const DDLIssueTemplate = "bb.issue.database.schema.update";
const DMLIssueTemplate = "bb.issue.database.data.update";

const router = useRouter();
const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const { pageMode } = storeToRefs(useActuatorV1Store());

const statement = computed(() => {
  const tab = tabStore.currentTab;
  return tab?.selectedStatement || tab?.statement || "";
});

const isDDL = computedAsync(async () => {
  const { data } = await parseSQL(statement.value);
  return data !== null ? isDDLStatement(data, "some") : false;
}, false);

const showActionButtons = computed(() => pageMode.value === "BUNDLED");

const handleClose = () => {
  emit("close");
};

const gotoCreateIssue = () => {
  const database = tabStore.currentTab?.connection.database ?? "";
  if (!database) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-editor.goto-edit-schema-hint"),
    });
    return;
  }

  emit("close");

  const db = useDatabaseV1Store().getDatabaseByName(database);

  router.push({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(db.project),
      issueSlug: "create",
    },
    query: {
      template: isDDL.value ? DDLIssueTemplate : DMLIssueTemplate,
      name: `[${db.databaseName}] ${
        isDDL.value ? "Alter schema" : "Change Data"
      }`,
      project: db.projectEntity.uid,
      databaseList: db.uid,
      sql: statement.value,
    },
  });
};
</script>
