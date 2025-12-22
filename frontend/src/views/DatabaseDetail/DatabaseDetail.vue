<template>
  <div
    class="flex-1 overflow-auto focus:outline-hidden flex flex-col gap-y-4"
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
        class="gap-y-2 flex flex-col items-start xl:flex-row xl:items-center xl:justify-between xl:gap-x-2"
      >
        <div class="flex-1 flex flex-col min-w-0 shrink-0 gap-y-2">
          <!-- Summary -->
          <div class="w-full flex flex-col">
            <div
              class="text-xl box-content font-bold text-main truncate flex items-center gap-x-2"
            >
              {{ database.databaseName }}

              <ProductionEnvironmentV1Icon
                :environment="environment"
                :tooltip="true"
                class="w-5 h-5"
              />
            </div>
            <div
              class="w-full flex flex-row items-center justify-start gap-x-1"
            >
              <EllipsisText class="textinfolabel">
                {{ database.name }}
              </EllipsisText>
              <CopyButton :content="database.name" />
            </div>
          </div>
          <dl
            class="flex flex-col gap-y-1 md:flex-row md:flex-wrap"
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
            <dt v-if="database.schemaVersion" class="sr-only">
              {{ $t("common.version") }}
            </dt>
            <dd
              v-if="database.schemaVersion"
              class="flex items-center text-sm md:mr-4"
            >
              <span class="ml-1 textlabel"
                >{{ $t("common.version") }}&nbsp;-&nbsp;</span
              >
              <span>{{ database.schemaVersion }}</span>
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
          <ExportSchemaButton :database="database" />
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
            v-if="allowChangeDatabase"
            @click="() => {
              preCreateIssue(database.project, [database.name])
            }"
          >
            <span>{{ $t("database.change-database") }}</span>
          </NButton>
        </div>
      </div>
    </main>

    <NTabs v-if="ready" v-model:value="state.selectedTab">
      <NTabPane name="overview" :tab="$t('common.overview')">
        <DatabaseOverviewPanel class="mt-2" :database="database" />
      </NTabPane>
      <NTabPane
        v-if="allowListChangelogs"
        name="changelog"
        :tab="$t('common.changelog')"
      >
        <DatabaseChangelogPanel class="mt-2" :database="database" />
      </NTabPane>
      <NTabPane name="revision" :tab="$t('database.revision.self')">
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
      <div class="pt-6 flex justify-end gap-x-2">
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
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { ArrowRightLeftIcon } from "lucide-vue-next";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { computed, reactive, watch, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBAttention, BBModal } from "@/bbkit";
import { useDatabaseDetailContext } from "@/components/Database/context";
import DatabaseChangelogPanel from "@/components/Database/DatabaseChangelogPanel.vue";
import DatabaseOverviewPanel from "@/components/Database/DatabaseOverviewPanel.vue";
import DatabaseRevisionPanel from "@/components/Database/DatabaseRevisionPanel.vue";
import DatabaseSensitiveDataPanel from "@/components/Database/DatabaseSensitiveDataPanel.vue";
import {
  DatabaseSettingsPanel,
  SchemaDiagramButton,
  SQLEditorButtonV1,
} from "@/components/DatabaseDetail";
import DriftedDatabaseAlert from "@/components/DatabaseDetail/DriftedDatabaseAlert.vue";
import ExportSchemaButton from "@/components/DatabaseDetail/ExportSchemaButton.vue";
import SyncDatabaseButton from "@/components/DatabaseDetail/SyncDatabaseButton.vue";
import EllipsisText from "@/components/EllipsisText.vue";
import { preCreateIssue } from "@/components/Plan/logic/issue";
import TransferOutDatabaseForm from "@/components/TransferOutDatabaseForm";
import {
  CopyButton,
  Drawer,
  EnvironmentV1Name,
  InstanceV1Name,
  ProductionEnvironmentV1Icon,
} from "@/components/v2";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1ByName, useEnvironmentV1Store } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import {
  extractProjectResourceName,
  instanceV1HasAlterSchema,
  isDatabaseV1Queryable,
} from "@/utils";

const databaseHashList = [
  "overview",
  "changelog",
  "revision",
  "setting",
  "catalog",
] as const;
export type DatabaseHash = (typeof databaseHashList)[number];
const isDatabaseHash = (x: unknown): x is DatabaseHash =>
  databaseHashList.includes(x as DatabaseHash);

interface LocalState {
  showTransferDatabaseModal: boolean;
  showIncorrectProjectModal: boolean;
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

const allowChangeDatabase = computed(() => {
  return allowChangeData.value || allowAlterSchema.value;
});

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
  state.showTransferDatabaseModal = true;
};

const handleGotoSQLEditorFailed = () => {
  state.showIncorrectProjectModal = true;
};

const environment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    database.value.effectiveEnvironment ?? ""
  );
});

useTitle(computed(() => database.value.databaseName));
</script>
