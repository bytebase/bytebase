<template>
  <div class="execute-hint w-112">
    <NAlert type="info">
      <section class="space-y-2">
        <p>
          <i18n-t keypath="sql-editor.only-select-allowed">
            <template #select>
              <strong
                ><code>SELECT</code>, <code>SHOW</code> and
                <code>SET</code></strong
              >
            </template>
          </i18n-t>
        </p>
        <p v-if="descriptions.action && descriptions.reaction">
          <i18n-t keypath="sql-editor.want-to-action">
            <template #want>
              {{ descriptions.want }}
            </template>
            <template #action>
              <strong>
                {{ descriptions.action }}
              </strong>
            </template>
            <template #reaction>
              {{ descriptions.reaction }}
            </template>
          </i18n-t>
        </p>
      </section>
    </NAlert>

    <div class="execute-hint-content mt-4 flex justify-between">
      <div
        v-if="actions.action && actions.admin"
        class="flex justify-start items-center space-x-2"
      >
        <AdminModeButton @enter="$emit('close')" />
      </div>
      <div class="flex flex-1 justify-end items-center space-x-2">
        <NButton @click="handleClose">{{ $t("common.close") }}</NButton>
        <NButton
          v-if="actions.action"
          type="primary"
          @click="handleClickActionButton"
        >
          <template v-if="actions.action === 'CREATE_ISSUE'">
            {{
              isDDL ? $t("database.edit-schema") : $t("database.change-data")
            }}
          </template>
          <template v-if="actions.action === 'STANDARD_MODE'">
            {{ $t("sql-editor.standard-mode.self") }}
          </template>
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
  useCurrentUserV1,
  useDatabaseV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { extractProjectResourceName, hasWorkspacePermissionV2 } from "@/utils";
import { useSQLEditorContext } from "../context";
import AdminModeButton from "./AdminModeButton.vue";

const emit = defineEmits<{
  (e: "close"): void;
}>();

const DDLIssueTemplate = "bb.issue.database.schema.update";
const DMLIssueTemplate = "bb.issue.database.data.update";

const router = useRouter();
const { t } = useI18n();
const me = useCurrentUserV1();
const tabStore = useSQLEditorTabStore();
const { standardModeEnabled } = useSQLEditorContext();
const { pageMode } = storeToRefs(useActuatorV1Store());

const statement = computed(() => {
  const tab = tabStore.currentTab;
  return tab?.selectedStatement || tab?.statement || "";
});

const isDDL = computedAsync(async () => {
  const { data } = await parseSQL(statement.value);
  return data !== null ? isDDLStatement(data, "some") : false;
}, false);

const actions = computed(() => {
  type Actions = {
    admin: boolean;
    action: "STANDARD_MODE" | "CREATE_ISSUE" | undefined;
  };
  const actions: Actions = {
    admin: false,
    action: undefined,
  };
  if (hasWorkspacePermissionV2(me.value, "bb.instances.adminExecute")) {
    actions.admin = true;
  }
  if (standardModeEnabled.value) {
    actions.action = "STANDARD_MODE";
  } else if (pageMode.value === "BUNDLED") {
    actions.action = "CREATE_ISSUE";
  }

  return actions;
});

const descriptions = computed(() => {
  const descriptions = {
    want: isDDL.value
      ? t("database.edit-schema").toLowerCase()
      : t("database.change-data").toLowerCase(),
    action: "",
    reaction: "",
  };
  const { admin, action } = actions.value;
  if (action === "CREATE_ISSUE") {
    descriptions.action = isDDL
      ? t("database.edit-schema")
      : t("database.change-data");
    descriptions.reaction = t("sql-editor.and-submit-an-issue");
  } else if (action === "STANDARD_MODE") {
    descriptions.action = t("sql-editor.standard-mode.self");
    descriptions.reaction = t("sql-editor.to-enable-standard-mode");
  } else if (admin) {
    descriptions.action = t("sql-editor.admin-mode.self");
    descriptions.reaction = t("sql-editor.to-enable-admin-mode");
  }
  return descriptions;
});

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
        isDDL.value ? "Edit schema" : "Change Data"
      }`,
      databaseList: db.name,
      sql: statement.value,
    },
  });
};

const changeToStandardMode = () => {
  const tab = tabStore.currentTab;
  if (!tab) {
    return;
  }
  if (tab.mode === "ADMIN") {
    return;
  }
  tab.mode = "STANDARD";
  emit("close");
};

const handleClickActionButton = () => {
  const { action } = actions.value;
  if (action === "CREATE_ISSUE") {
    gotoCreateIssue();
    return;
  }
  if (action === "STANDARD_MODE") {
    changeToStandardMode();
    return;
  }
};
</script>
