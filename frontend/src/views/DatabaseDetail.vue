<template>
  <div
    class="flex-1 overflow-auto focus:outline-none p-6 space-y-4"
    tabindex="0"
    v-bind="$attrs"
  >
    <main class="flex-1 relative">
      <!-- Highlight Panel -->
      <div
        class="gap-y-2 flex flex-col items-start lg:flex-row lg:items-center lg:justify-between"
      >
        <div class="flex-1 min-w-0 shrink-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center">
                <h1
                  class="pb-2.5 text-xl font-bold leading-6 text-main truncate flex items-center gap-x-3"
                >
                  {{ database.databaseName }}

                  <ProductionEnvironmentV1Icon
                    :environment="environment"
                    :tooltip="true"
                    class="w-5 h-5"
                  />

                  <BBBadge
                    v-if="isPITRDatabaseV1(database)"
                    text="PITR"
                    :can-remove="false"
                    class="text-xs"
                  />
                </h1>
              </div>
            </div>
          </div>
          <dl
            class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
            data-label="bb-database-detail-info-block"
          >
            <dt class="sr-only">{{ $t("common.environment") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.environment") }}&nbsp;-&nbsp;</span
              >
              <EnvironmentV1Name
                :environment="environment"
                icon-class="textinfolabel"
              />
            </dd>
            <dt class="sr-only">{{ $t("common.instance") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="ml-1 textlabel"
                >{{ $t("common.instance") }}&nbsp;-&nbsp;</span
              >
              <InstanceV1Name :instance="database.instanceEntity" />
            </dd>
            <dt class="sr-only">{{ $t("common.project") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.project") }}&nbsp;-&nbsp;</span
              >
              <ProjectV1Name
                :project="database.projectEntity"
                hash="#databases"
              />
            </dd>
            <SQLEditorButtonV1
              class="text-sm md:mr-4"
              :database="database"
              :label="true"
              :disabled="!allowQuery"
              @failed="handleGotoSQLEditorFailed"
            />
            <dd
              v-if="hasSchemaDiagramFeature"
              class="flex items-center text-sm md:mr-4 textlabel cursor-pointer hover:text-accent"
              @click.prevent="state.showSchemaDiagram = true"
            >
              <span class="mr-1">{{ $t("schema-diagram.self") }}</span>
              <SchemaDiagramIcon />
            </dd>
          </dl>
        </div>
        <div
          v-if="allowToChangeDatabase"
          class="flex flex-row justify-start items-center flex-wrap shrink gap-x-2 gap-y-2"
          data-label="bb-database-detail-action-buttons-container"
        >
          <BBSpin v-if="state.syncingSchema" :title="$t('instance.syncing')" />
          <NButton
            :disabled="state.syncingSchema"
            @click.prevent="syncDatabaseSchema"
          >
            {{ $t("common.sync-now") }}
          </NButton>
          <NButton
            v-if="allowTransferProject"
            @click.prevent="tryTransferProject"
          >
            <span>{{ $t("database.transfer-project") }}</span>
            <heroicons-outline:switch-horizontal
              class="-mr-1 ml-2 h-5 w-5 text-control-light"
            />
          </NButton>
          <NButton
            v-if="allowAlterSchemaOrChangeData"
            @click="createMigration('bb.issue.database.data.update')"
          >
            <span>{{ $t("database.change-data") }}</span>
          </NButton>
          <NButton
            v-if="allowAlterSchema"
            @click="createMigration('bb.issue.database.schema.update')"
          >
            <span>{{ $t("database.edit-schema") }}</span>
          </NButton>
        </div>
      </div>
    </main>

    <NTabs v-model:value="state.selectedTab">
      <NTabPane name="overview" :tab="$t('common.overview')">
        <DatabaseOverviewPanel :database="database" />
      </NTabPane>
      <NTabPane
        v-if="allowToChangeDatabase"
        name="change-history"
        :tab="$t('change-history.self')"
      >
        <DatabaseChangeHistoryPanel
          :database="database"
          :allow-edit="allowEdit"
        />
      </NTabPane>
      <NTabPane
        v-if="
          allowToChangeDatabase &&
          instanceV1HasBackupRestore(database.instanceEntity)
        "
        name="backup-and-restore"
        :tab="$t('common.backup-and-restore')"
      >
        <DatabaseBackupPanel
          :database="database"
          :allow-admin="allowAdmin"
          :allow-edit="allowEdit"
        />
      </NTabPane>
      <NTabPane
        v-if="
          allowToChangeDatabase &&
          instanceV1SupportSlowQuery(database.instanceEntity)
        "
        name="slow-query"
        :tab="$t('slow-query.slow-queries')"
      >
        <DatabaseSlowQueryPanel :database="database" />
      </NTabPane>
      <NTabPane
        v-if="allowToChangeDatabase"
        name="setting"
        :tab="$t('common.settings')"
      >
        <DatabaseSettingsPanel :database="database" :allow-edit="allowEdit" />
      </NTabPane>
    </NTabs>

    <TransferSingleDatabase
      v-if="state.showTransferDatabaseModal"
      :database="database"
      @cancel="state.showTransferDatabaseModal = false"
      @updated="state.showTransferDatabaseModal = false"
    />

    <BBModal
      v-if="state.showIncorrectProjectModal"
      :title="$t('common.warning')"
      @close="state.showIncorrectProjectModal = false"
    >
      <div class="col-span-1 w-96">
        {{ $t("database.incorrect-project-warning") }}
      </div>
      <div class="pt-6 flex justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="state.showIncorrectProjectModal = false"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click.prevent="
            state.showIncorrectProjectModal = false;
            state.showTransferDatabaseModal = true;
          "
        >
          {{ $t("database.go-to-transfer") }}
        </button>
      </div>
    </BBModal>
  </div>

  <BBModal
    v-if="state.showSchemaDiagram"
    :title="$t('schema-diagram.self')"
    class="h-[calc(100vh-40px)] !max-h-[calc(100vh-40px)]"
    header-class="!border-0"
    container-class="flex-1 !pt-0"
    @close="state.showSchemaDiagram = false"
  >
    <div class="w-[80vw] h-full">
      <SchemaDiagram
        :database="database"
        :database-metadata="dbSchemaStore.getDatabaseMetadata(database.name)"
      />
    </div>
  </BBModal>

  <SchemaEditorModal
    v-if="state.showSchemaEditorModal"
    :database-id-list="[database.uid]"
    alter-type="SINGLE_DB"
    @close="state.showSchemaEditorModal = false"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { ClientError } from "nice-grpc-web";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useRoute } from "vue-router";
import DatabaseBackupPanel from "@/components/DatabaseBackupPanel.vue";
import DatabaseChangeHistoryPanel from "@/components/DatabaseChangeHistoryPanel.vue";
import {
  DatabaseSettingsPanel,
  SQLEditorButtonV1,
} from "@/components/DatabaseDetail";
import DatabaseOverviewPanel from "@/components/DatabaseOverviewPanel.vue";
import DatabaseSlowQueryPanel from "@/components/DatabaseSlowQueryPanel.vue";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { TransferSingleDatabase } from "@/components/TransferDatabaseForm";
import {
  EnvironmentV1Name,
  InstanceV1Name,
  ProductionEnvironmentV1Icon,
  ProjectV1Name,
} from "@/components/v2";
import {
  pushNotification,
  useCurrentUserIamPolicy,
  useCurrentUserV1,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useEnvironmentV1Store,
} from "@/store";
import {
  UNKNOWN_ID,
  DEFAULT_PROJECT_V1_NAME,
  unknownEnvironment,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import {
  idFromSlug,
  hasWorkspacePermissionV1,
  isPITRDatabaseV1,
  isArchivedDatabaseV1,
  instanceV1HasBackupRestore,
  instanceV1SupportSlowQuery,
  hasPermissionInProjectV1,
  instanceV1HasAlterSchema,
  isDatabaseV1Queryable,
  allowUsingSchemaEditorV1,
} from "@/utils";

const databaseHashList = [
  "overview",
  "change-history",
  "backup-and-restore",
  "slow-query",
  "setting",
] as const;
export type DatabaseHash = typeof databaseHashList[number];
const isDatabaseHash = (x: any): x is DatabaseHash =>
  databaseHashList.includes(x);

interface LocalState {
  showTransferDatabaseModal: boolean;
  showIncorrectProjectModal: boolean;
  showSchemaEditorModal: boolean;
  currentProjectId: string;
  selectedIndex: number;
  syncingSchema: boolean;
  showSchemaDiagram: boolean;
  selectedTab: DatabaseHash;
}

const props = defineProps({
  databaseSlug: {
    required: true,
    type: String,
  },
});

const { t } = useI18n();
const router = useRouter();
const databaseV1Store = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();

const state = reactive<LocalState>({
  showTransferDatabaseModal: false,
  showIncorrectProjectModal: false,
  showSchemaEditorModal: false,
  currentProjectId: String(UNKNOWN_ID),
  selectedIndex: 0,
  syncingSchema: false,
  showSchemaDiagram: false,
  selectedTab: "overview",
});
const route = useRoute();
const currentUserV1 = useCurrentUserV1();
const currentUserIamPolicy = useCurrentUserIamPolicy();

watch(
  () => route.hash,
  (hash) => {
    let targetHash = hash.replace(/^#?/g, "") as DatabaseHash;
    if (isDatabaseHash(targetHash)) {
      state.selectedTab = targetHash;
    }
  },
  { immediate: true }
);

watch(
  () => state.selectedTab,
  (tab) => {
    router.replace({
      name: "workspace.database.detail",
      hash: `#${tab}`,
      query: route.query,
    });
  }
);

const database = computed(() => {
  return databaseV1Store.getDatabaseByUID(
    String(idFromSlug(props.databaseSlug))
  );
});
const project = computed(() => database.value.projectEntity);

const allowToChangeDatabase = computed(() => {
  return currentUserIamPolicy.allowToChangeDatabaseOfProject(
    project.value.name
  );
});

const hasSchemaDiagramFeature = computed((): boolean => {
  return instanceV1HasAlterSchema(database.value.instanceEntity);
});

const allowQuery = computed(() => {
  return isDatabaseV1Queryable(database.value, currentUserV1.value);
});

// Project can be transferred if meets either of the condition below:
// - Database is in default project
// - Workspace role can manage instance
// - Project role can transfer database
const allowTransferProject = computed(() => {
  if (isArchivedDatabaseV1(database.value)) {
    return false;
  }

  if (database.value.project === DEFAULT_PROJECT_V1_NAME) {
    return true;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-project",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.transfer-database"
    )
  ) {
    return true;
  }

  return false;
});

// Database can be admined if meets either of the condition below:
// - Workspace role can manage instance
// - Project role can admin database
//
// The admin operation includes
// - Edit database label
// - Enable/disable backup
const allowAdmin = computed(() => {
  if (isArchivedDatabaseV1(database.value)) {
    return false;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.admin-database"
    )
  ) {
    return true;
  }
  return false;
});

// Database can be edited if meets either of the condition below:
// - Workspace role can manage instance
// - Project role can change database
//
// The edit operation includes
// - Take manual backup
const allowEdit = computed(() => {
  if (isArchivedDatabaseV1(database.value)) {
    return false;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.change-database"
    )
  ) {
    return true;
  }
  return false;
});

const allowAlterSchemaOrChangeData = computed(() => {
  if (database.value.project === DEFAULT_PROJECT_V1_NAME) {
    return false;
  }
  return allowEdit.value;
});

const allowAlterSchema = computed(() => {
  return (
    allowAlterSchemaOrChangeData.value &&
    instanceV1HasAlterSchema(database.value.instanceEntity)
  );
});

const tryTransferProject = () => {
  state.currentProjectId = project.value.uid;
  state.showTransferDatabaseModal = true;
};

const createMigration = async (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update"
) => {
  if (type === "bb.issue.database.schema.update") {
    if (
      database.value.syncState === State.ACTIVE &&
      allowUsingSchemaEditorV1([database.value])
    ) {
      state.showSchemaEditorModal = true;
      return;
    }
  }

  // Create a user friendly default issue name
  const issueNameParts: string[] = [];
  issueNameParts.push(`[${database.value.databaseName}]`);
  issueNameParts.push(
    type === "bb.issue.database.schema.update" ? `Alter schema` : `Change data`
  );
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  const query: Record<string, any> = {
    template: type,
    name: issueNameParts.join(" "),
    project: project.value.uid,
    databaseList: database.value.uid,
  };

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  });
};

const handleGotoSQLEditorFailed = () => {
  state.currentProjectId = database.value.projectEntity.uid;
  state.showIncorrectProjectModal = true;
};

const syncDatabaseSchema = async () => {
  state.syncingSchema = true;

  try {
    await databaseV1Store.syncDatabase(database.value.name);

    dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.value.name,
      skipCache: true,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC,
    });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t(
        "db.successfully-synced-schema-for-database-database-value-name",
        [database.value.databaseName]
      ),
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema-for-database-database-value-name", [
        database.value.databaseName,
      ]),
      description: (error as ClientError).details,
    });
  } finally {
    state.syncingSchema = false;
  }
};

const environment = computed(() => {
  return (
    useEnvironmentV1Store().getEnvironmentByName(
      database.value.effectiveEnvironment
    ) ?? unknownEnvironment()
  );
});
</script>
