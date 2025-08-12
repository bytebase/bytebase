<template>
  <div
    class="flex-1 overflow-auto focus:outline-none space-y-4"
    tabindex="0"
    v-bind="$attrs"
  >
    <BBAttention
      v-if="!database.effectiveEnvironment"
      class="w-full mb-4"
      :type="'warning'"
      :action-text="$t('database.set-environment')"
      @click="
        () => {
          state.selectedTab = 'setting';
        }
      "
    >
      {{ $t("database.no-environment") }}
    </BBAttention>
    <DriftedDatabaseAlert :database="database" />

    <main class="flex-1 relative">
      <!-- Highlight Panel -->
      <div
        class="gap-y-2 flex flex-col items-start xl:flex-row xl:items-center xl:justify-between"
      >
        <div class="flex-1 min-w-0 shrink-0">
          <!-- Summary -->
          <div class="w-full flex items-center">
            <div class="w-full flex items-baseline gap-x-2">
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
              <div class="flex items-center space-x-1">
                <span class="textinfolabel">
                  {{ database.name }}
                </span>
                <CopyButton :content="database.name" />
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
            icon-placement="right"
          >
            <span>{{ $t("database.transfer-project") }}</span>
            <template #icon>
              <ArrowRightLeftIcon :size="16" />
            </template>
          </NButton>
          <NButton
            v-if="allowChangeData"
            :disabled="!database.effectiveEnvironment"
            @click="createMigration('bb.issue.database.data.update')"
          >
            <span>{{ $t("database.change-data") }}</span>
          </NButton>
          <NButton
            v-if="allowAlterSchema"
            :disabled="!database.effectiveEnvironment"
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
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import dayjs from "dayjs";
import { ArrowRightLeftIcon } from "lucide-vue-next";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { computed, reactive, watch, watchEffect } from "vue";
import { useRouter, useRoute, type LocationQueryRaw } from "vue-router";
import { BBModal } from "@/bbkit";
import { BBAttention } from "@/bbkit";
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
import DriftedDatabaseAlert from "@/components/DatabaseDetail/DriftedDatabaseAlert.vue";
import SyncDatabaseButton from "@/components/DatabaseDetail/SyncDatabaseButton.vue";
import TransferOutDatabaseForm from "@/components/TransferOutDatabaseForm";
import { Drawer } from "@/components/v2";
import {
  EnvironmentV1Name,
  InstanceV1Name,
  ProductionEnvironmentV1Icon,
} from "@/components/v2";
import { CopyButton } from "@/components/v2";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  useAppFeature,
  useEnvironmentV1Store,
  useDatabaseV1ByName,
} from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { UNKNOWN_PROJECT_NAME } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
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
  currentProjectName: string;
  selectedIndex: number;
  selectedTab: DatabaseHash;
}

const props = defineProps<{
  projectId: string;
  instanceId: string;
  databaseName: string;
}>();

const router = useRouter();

const state = reactive<LocalState>({
  showTransferDatabaseModal: false,
  showIncorrectProjectModal: false,
  showSchemaEditorModal: false,
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

watchEffect(() => {
  if (!ready.value) {
    return;
  }
  if (extractProjectResourceName(project.value.name) === props.projectId) {
    return;
  }
  router.replace({
    name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      instanceId: props.instanceId,
      databaseName: props.databaseName,
    },
    hash: `#${state.selectedTab}`,
    query: route.query,
  });
});

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

  const query: LocationQueryRaw = {
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
  return useEnvironmentV1Store().getEnvironmentByName(
    database.value.effectiveEnvironment ?? ""
  );
});

useTitle(computed(() => database.value.databaseName));
</script>
