<template>
  <div
    class="flex-1 overflow-auto focus:outline-none space-y-4"
    tabindex="0"
    v-bind="$attrs"
  >
    <NAlert
      v-if="database.drifted"
      type="warning"
      :title="$t('database.drifted.schema-drift-detected.self')"
    >
      <div class="flex items-center justify-between gap-4">
        <div class="flex-1">
          {{ $t("database.drifted.schema-drift-detected.description") }}
          <a
            href="https://docs.bytebase.com/change-database/drift-detection/?source=console"
            target="_blank"
            class="text-accent hover:underline ml-1"
          >
            {{ $t("common.learn-more") }}
          </a>
        </div>
        <div class="flex justify-end items-center gap-2">
          <NButton size="small" @click="state.showSchemaDiffModal = true">
            {{ $t("database.drifted.view-diff") }}
          </NButton>
          <NButton
            v-if="database.project !== DEFAULT_PROJECT_NAME"
            size="small"
            type="primary"
            @click="updateDatabaseDrift"
          >
            {{ $t("database.drifted.new-baseline.self") }}
          </NButton>
        </div>
      </div>
    </NAlert>

    <main class="flex-1 relative">
      <!-- Highlight Panel -->
      <div
        class="gap-y-2 flex flex-col items-start lg:flex-row lg:items-center lg:justify-between"
      >
        <div class="flex-1 min-w-0 shrink-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-baseline gap-x-2">
                <h1
                  class="text-xl font-bold text-main truncate flex items-center gap-x-2"
                >
                  {{ database.databaseName }}

                  <ProductionEnvironmentV1Icon
                    :environment="environment"
                    :tooltip="true"
                    class="w-5 h-5"
                  />
                </h1>
                <div class="flex items-center">
                  <span class="textinfolabel">
                    {{ database.name }}
                  </span>
                  <NButton
                    v-if="isSupported"
                    quaternary
                    size="tiny"
                    @click="handleCopyDatabaseName(database.name)"
                  >
                    <ClipboardCopyIcon class="w-4 h-4" />
                  </NButton>
                </div>
              </div>
            </div>
          </div>
          <dl
            class="flex flex-col space-y-1 md:flex-row md:flex-wrap"
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
              <InstanceV1Name :instance="database.instanceResource" />
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
              v-if="allowQuery"
              class="text-sm md:mr-4"
              :database="database"
              :label="true"
              @failed="handleGotoSQLEditorFailed"
            />
            <SchemaDiagramButton
              v-if="hasSchemaDiagramFeature"
              class="md:mr-4"
              :database="database"
            />
          </dl>
        </div>
        <div
          class="flex flex-row justify-start items-center flex-wrap shrink gap-x-2 gap-y-2"
          data-label="bb-database-detail-action-buttons-container"
        >
          <SyncDatabaseButton
            v-if="allowSyncDatabase"
            :type="'default'"
            :text="false"
            :database="database"
          />
          <NButton
            v-if="allowTransferDatabase"
            @click.prevent="tryTransferProject"
          >
            <span>{{ $t("database.transfer-project") }}</span>
            <ArrowRightLeftIcon class="ml-1" :size="16" />
          </NButton>
          <NButton
            v-if="allowChangeData"
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

    <NTabs v-if="ready" v-model:value="state.selectedTab">
      <NTabPane name="overview" :tab="$t('common.overview')">
        <DatabaseOverviewPanel class="mt-2" :database="database" />
      </NTabPane>
      <NTabPane
        v-if="
          databaseChangeMode === DatabaseChangeMode.PIPELINE &&
          allowListChangelogs
        "
        name="changelog"
        :tab="$t('common.changelog')"
      >
        <DatabaseChangelogPanel class="mt-2" :database="database" />
      </NTabPane>
      <NTabPane
        v-if="databaseChangeMode === DatabaseChangeMode.PIPELINE"
        name="revision"
        :tab="$t('database.revision.self')"
      >
        <DatabaseRevisionPanel class="mt-2" :database="database" />
      </NTabPane>
      <NTabPane name="catalog" :tab="$t('common.catalog')">
        <DatabaseSensitiveDataPanel class="mt-2" :database="database" />
      </NTabPane>
      <NTabPane
        v-if="allowUpdateDatabase"
        name="setting"
        :tab="$t('common.settings')"
      >
        <DatabaseSettingsPanel class="mt-2" :database="database" />
      </NTabPane>
    </NTabs>

    <BBModal
      v-if="state.showIncorrectProjectModal"
      :title="$t('common.warning')"
      @close="state.showIncorrectProjectModal = false"
    >
      <div class="col-span-1 w-96">
        {{ $t("database.incorrect-project-warning") }}
      </div>
      <div class="pt-6 flex justify-end space-x-3">
        <NButton @click.prevent="state.showIncorrectProjectModal = false">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          @click.prevent="
            state.showIncorrectProjectModal = false;
            state.showTransferDatabaseModal = true;
          "
        >
          {{ $t("database.go-to-transfer") }}
        </NButton>
      </div>
    </BBModal>
  </div>

  <Drawer
    :show="state.showTransferDatabaseModal"
    :auto-focus="true"
    @close="state.showTransferDatabaseModal = false"
  >
    <TransferOutDatabaseForm
      :database-list="[database]"
      :selected-database-names="[database.name]"
      @dismiss="state.showTransferDatabaseModal = false"
    />
  </Drawer>

  <SchemaEditorModal
    v-if="state.showSchemaEditorModal"
    :database-names="[database.name]"
    alter-type="SINGLE_DB"
    @close="state.showSchemaEditorModal = false"
  />

  <BBModal
    v-if="state.showSchemaDiffModal"
    :title="$t('database.drifted.view-diff')"
    header-class="!border-0"
    container-class="!pt-0"
    @close="state.showSchemaDiffModal = false"
  >
    <div
      style="width: calc(100vw - 9rem); height: calc(100vh - 10rem)"
      class="relative"
    >
      <DiffEditor
        class="w-full h-full border rounded-md overflow-clip"
        :original="latestChangelogSchema"
        :modified="currentDatabaseSchema"
        :readonly="true"
        language="sql"
      />
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { useClipboard } from "@vueuse/core";
import dayjs from "dayjs";
import { ArrowRightLeftIcon, ClipboardCopyIcon } from "lucide-vue-next";
import { NAlert, NButton, NTabPane, NTabs } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, useRoute } from "vue-router";
import { BBModal } from "@/bbkit";
import SchemaEditorModal from "@/components/AlterSchemaPrepForm/SchemaEditorModal.vue";
import DatabaseChangelogPanel from "@/components/Database/DatabaseChangelogPanel.vue";
import DatabaseOverviewPanel from "@/components/Database/DatabaseOverviewPanel.vue";
import DatabaseRevisionPanel from "@/components/Database/DatabaseRevisionPanel.vue";
import DatabaseSensitiveDataPanel from "@/components/Database/DatabaseSensitiveDataPanel.vue";
import { useDatabaseDetailContext } from "@/components/Database/context";
import {
  DatabaseSettingsPanel,
  SQLEditorButtonV1,
  SchemaDiagramButton,
} from "@/components/DatabaseDetail";
import SyncDatabaseButton from "@/components/DatabaseDetail/SyncDatabaseButton.vue";
import { DiffEditor } from "@/components/MonacoEditor";
import TransferOutDatabaseForm from "@/components/TransferOutDatabaseForm";
import { Drawer } from "@/components/v2";
import {
  EnvironmentV1Name,
  InstanceV1Name,
  ProductionEnvironmentV1Icon,
  ProjectV1Name,
} from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useAppFeature,
  useEnvironmentV1Store,
  useDatabaseV1ByName,
  pushNotification,
  useDatabaseV1Store,
  useChangelogStore,
} from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import {
  UNKNOWN_PROJECT_NAME,
  unknownEnvironment,
  DEFAULT_PROJECT_NAME,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { ChangelogView } from "@/types/proto/v1/database_service";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import {
  instanceV1HasAlterSchema,
  isDatabaseV1Queryable,
  allowUsingSchemaEditor,
  extractProjectResourceName,
} from "@/utils";

const databaseHashList = [
  "overview",
  "changelog",
  "revision",
  "setting",
  "catalog",
] as const;
export type DatabaseHash = (typeof databaseHashList)[number];
const isDatabaseHash = (x: any): x is DatabaseHash =>
  databaseHashList.includes(x);

interface LocalState {
  showTransferDatabaseModal: boolean;
  showIncorrectProjectModal: boolean;
  showSchemaEditorModal: boolean;
  showSchemaDiffModal: boolean;
  currentProjectName: string;
  selectedIndex: number;
  selectedTab: DatabaseHash;
}

const props = defineProps<{
  projectId: string;
  instanceId: string;
  databaseName: string;
}>();

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const changelogStore = useChangelogStore();

const state = reactive<LocalState>({
  showTransferDatabaseModal: false,
  showIncorrectProjectModal: false,
  showSchemaEditorModal: false,
  showSchemaDiffModal: false,
  currentProjectName: UNKNOWN_PROJECT_NAME,
  selectedIndex: 0,
  selectedTab: "overview",
});
const route = useRoute();
const {
  allowSyncDatabase,
  allowUpdateDatabase,
  allowTransferDatabase,
  allowChangeData,
  allowAlterSchema,
  allowListChangelogs,
} = useDatabaseDetailContext();
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

watch(
  () => route.hash,
  (hash) => {
    const targetHash = hash.replace(/^#?/g, "") as DatabaseHash;
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
      hash: `#${tab}`,
      query: route.query,
    });
  },
  { immediate: true }
);

const { database, ready } = useDatabaseV1ByName(
  computed(
    () =>
      `${instanceNamePrefix}${props.instanceId}/${databaseNamePrefix}${props.databaseName}`
  )
);

const project = computed(() => database.value.projectEntity);

const hasSchemaDiagramFeature = computed((): boolean => {
  return instanceV1HasAlterSchema(database.value.instanceResource);
});

const allowQuery = computed(() => {
  return isDatabaseV1Queryable(database.value);
});

const tryTransferProject = () => {
  state.currentProjectName = project.value.name;
  state.showTransferDatabaseModal = true;
};

const createMigration = async (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update"
) => {
  if (type === "bb.issue.database.schema.update") {
    if (
      database.value.state === State.ACTIVE &&
      allowUsingSchemaEditor([database.value])
    ) {
      state.showSchemaEditorModal = true;
      return;
    }
  }

  // Create a user friendly default issue name
  const issueNameParts: string[] = [];
  issueNameParts.push(`[${database.value.databaseName}]`);
  issueNameParts.push(
    type === "bb.issue.database.schema.update" ? `Edit schema` : `Change data`
  );
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  const query: Record<string, any> = {
    template: type,
    name: issueNameParts.join(" "),
    databaseList: database.value.name,
  };

  router.push({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      issueSlug: "create",
    },
    query,
  });
};

const handleGotoSQLEditorFailed = () => {
  state.currentProjectName = database.value.project;
  state.showIncorrectProjectModal = true;
};

const environment = computed(() => {
  return (
    useEnvironmentV1Store().getEnvironmentByName(
      database.value.effectiveEnvironment
    ) ?? unknownEnvironment()
  );
});

useTitle(computed(() => database.value.databaseName));

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});
const handleCopyDatabaseName = (name: string) => {
  if (!isSupported.value) {
    return;
  }
  copyTextToClipboard(name).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
};

const updateDatabaseDrift = async () => {
  await databaseStore.updateDatabase({
    database: {
      ...database.value,
      drifted: false,
    },
    updateMask: ["drifted"],
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("database.drifted.new-baseline.successfully-established"),
  });
};

const latestChangelogSchema = ref("-- Loading latest changelog schema...");
const currentDatabaseSchema = ref("-- Loading current database schema...");

const fetchLatestChangelogSchema = async () => {
  try {
    const changelogs = await changelogStore.getOrFetchChangelogListOfDatabase(
      database.value.name,
      1, // Only get the latest one
      ChangelogView.CHANGELOG_VIEW_FULL
    );
    if (changelogs.length > 0) {
      const latestChangelog = changelogs[0];
      latestChangelogSchema.value =
        latestChangelog.schema || "-- No schema found in latest changelog";
    } else {
      latestChangelogSchema.value = "-- No changelogs found for this database";
    }
  } catch (error) {
    console.error("Failed to fetch changelog:", error);
    latestChangelogSchema.value = "-- Failed to load changelog schema";
  }
};

const fetchCurrentDatabaseSchema = async () => {
  try {
    const schemaResponse = await databaseStore.fetchDatabaseSchema(
      database.value.name
    );
    currentDatabaseSchema.value =
      schemaResponse.schema || "-- No schema available";
  } catch (error) {
    console.error("Failed to fetch database schema:", error);
    currentDatabaseSchema.value = "-- Failed to load database schema";
  }
};

// Fetch data when modal is opened
watch(
  () => state.showSchemaDiffModal,
  (show) => {
    if (show && database.value.drifted) {
      fetchLatestChangelogSchema();
      fetchCurrentDatabaseSchema();
    }
  }
);
</script>
