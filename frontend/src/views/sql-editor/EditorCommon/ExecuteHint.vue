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
                :environment="getDatabaseEnvironment(database)"
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
import { applyPlanTitleToQuery } from "@/components/Plan/logic/title";
import { EnvironmentV1Name } from "@/components/v2";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import {
  useDatabaseV1Store as databaseV1Store,
  useProjectV1Store as projectV1Store,
  pushNotification,
  useStorageStore as storageStoreAccessor,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import { unknownProject } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  getDatabaseEnvironment,
} from "@/utils";
import AdminModeButton from "./AdminModeButton.vue";

withDefaults(
  defineProps<{
    database?: Database | undefined;
  }>(),
  { database: undefined }
);

const emit = defineEmits<{
  (e: "close"): void;
}>();

const router = useRouter();
const { t } = useI18n();
const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();
// Store accessors imported under non-`use*` aliases so SonarCloud's
// React-hook-rule (typescript:S6440) doesn't misfire on these Pinia
// calls in a Vue SFC. Pinia stores are module-level singletons; the
// accessor name is cosmetic. Kept consistent across ExecuteHint.vue
// and SQLEditorHomePage.vue.
const databaseStore = databaseV1Store();
const projectStore = projectV1Store();
const storageStore = storageStoreAccessor();

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
  if (editorStore.allowAdmin) {
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

  const db = await databaseStore.getOrFetchDatabaseByName(database);
  // Project-fetch-failed cell: if the project lookup rejects (transient
  // network/permission failure), still open the plan page using the
  // already-known `db.project`. Fall back to `unknownProject()` which
  // has `enforceIssueTitle=true` — that is the safe governance default:
  // the launcher drops `query.name` so the plan page opens with a blank
  // title and the user types a deliberate one before submitting. Backend
  // remains the source of truth on submit.
  const project = await projectStore
    .getOrFetchProjectByName(db.project)
    .catch(() => unknownProject());
  const sqlStorageKey = `bb.issues.sql.${uuidv4()}`;
  storageStore.put(sqlStorageKey, statement.value);
  const { databaseName } = extractDatabaseResourceName(db.name);

  const query: Record<string, string> = {
    template: "bb.plan.change-database",
    databaseList: db.name,
    sqlStorageKey,
  };
  applyPlanTitleToQuery(
    query,
    project,
    () => `[${databaseName}] Change from SQL Editor`
  );

  const route = router.resolve({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: {
      projectId: extractProjectResourceName(db.project),
      planId: "create",
      specId: "placeholder",
    },
    query,
  });
  window.open(route.fullPath, "_blank");
};

const handleClickCreateIssue = () => {
  gotoCreateIssue();
};
</script>
