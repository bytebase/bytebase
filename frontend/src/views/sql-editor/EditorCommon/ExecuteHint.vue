<template>
  <div class="execute-hint w-112">
    <NAlert type="info">
      <section class="flex flex-col gap-y-2">
        <p>
          {{ $t("sql-editor.only-select-allowed") }}
        </p>
        <p v-if="database">
          <i18n-t keypath="sql-editor.enable-ddl-for-environment">
            <template #environment>
              <EnvironmentV1Name
                :environment="database.effectiveEnvironmentEntity"
              />
            </template>
          </i18n-t>
        </p>
      </section>
    </NAlert>

    <div class="execute-hint-content mt-4 flex justify-between">
      <div
        v-if="actions.issue && actions.admin"
        class="flex justify-start items-center gap-x-2"
      >
        <AdminModeButton @enter="$emit('close')" />
      </div>
      <div class="flex flex-1 justify-end items-center gap-x-2">
        <NButton @click="handleClose">{{ $t("common.close") }}</NButton>
        <NButton
          v-if="actions.issue"
          type="primary"
          @click="handleClickCreateIssue"
        >
          {{ descriptions.action }}
        </NButton>
        <AdminModeButton v-else @enter="$emit('close')" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NAlert, NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { EnvironmentV1Name } from "@/components/v2";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useDatabaseV1Store,
  useSQLEditorTabStore,
  useStorageStore,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { extractProjectResourceName, hasWorkspacePermissionV2 } from "@/utils";
import AdminModeButton from "./AdminModeButton.vue";

withDefaults(
  defineProps<{
    database?: ComposedDatabase | undefined;
  }>(),
  { database: undefined }
);

const emit = defineEmits<{
  (e: "close"): void;
}>();

const router = useRouter();
const { t } = useI18n();
const tabStore = useSQLEditorTabStore();

const statement = computed(() => {
  const tab = tabStore.currentTab;
  return tab?.selectedStatement || tab?.statement || "";
});

const actions = computed(() => {
  type Actions = {
    admin: boolean;
    issue: boolean;
  };
  const actions: Actions = {
    admin: false,
    issue: true,
  };
  if (hasWorkspacePermissionV2("bb.sql.admin")) {
    actions.admin = true;
  }

  return actions;
});

const descriptions = computed(() => {
  const descriptions = {
    want: t("database.change-database").toLowerCase(),
    action: "",
    reaction: "",
  };
  const { admin, issue } = actions.value;
  if (issue) {
    descriptions.action = t("database.change-database");
    descriptions.reaction = t("sql-editor.and-submit-an-issue");
  } else if (admin) {
    descriptions.action = t("sql-editor.admin-mode.self");
    descriptions.reaction = t("sql-editor.to-enable-admin-mode");
  }
  return descriptions;
});

const handleClose = () => {
  emit("close");
};

const gotoCreateIssue = async () => {
  const database = tabStore.currentTab?.connection.database ?? "";
  if (!database) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "No database selected",
    });
    return;
  }

  emit("close");

  const { enabledNewLayout } = useIssueLayoutVersion();
  const db = await useDatabaseV1Store().getOrFetchDatabaseByName(database);
  const sqlStorageKey = `bb.issues.sql.${uuidv4()}`;
  useStorageStore().put(sqlStorageKey, statement.value);

  const query = {
    template: "bb.issue.database.update",
    name: `[${db.databaseName}] Change from SQL Editor`,
    databaseList: db.name,
    sqlStorageKey,
  };

  const route = enabledNewLayout.value
    ? router.resolve({
        name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
        params: {
          projectId: extractProjectResourceName(db.project),
          planId: "create",
          specId: "placeholder",
        },
        query,
      })
    : router.resolve({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(db.project),
          issueSlug: "create",
        },
        query,
      });
  window.open(route.fullPath, "_blank");
};

const handleClickCreateIssue = () => {
  gotoCreateIssue();
};
</script>
