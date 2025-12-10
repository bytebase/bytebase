<template>
  <NAlert
    v-if="database.drifted"
    type="warning"
    :title="$t('database.drifted.schema-drift-detected.self')"
  >
    <div class="flex items-center justify-between gap-4">
      <div class="flex-1">
        {{ $t("database.drifted.schema-drift-detected.description") }}
        <LearnMoreLink
          class="ml-1"
          url="https://docs.bytebase.com/change-database/drift-detection/?source=console"
        />
      </div>
      <div class="flex justify-end items-center gap-x-2">
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

  <BBModal
    v-if="state.showSchemaDiffModal"
    :title="$t('database.drifted.view-diff')"
    header-class="border-0!"
    container-class="pt-0!"
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
import { create } from "@bufbuild/protobuf";
import { NAlert, NButton } from "naive-ui";
import { reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { DiffEditor } from "@/components/MonacoEditor";
import {
  pushNotification,
  useChangelogStore,
  useDatabaseV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import {
  ChangelogView,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";

interface LocalState {
  showSchemaDiffModal: boolean;
}

interface Props {
  database: ComposedDatabase;
}

const props = defineProps<Props>();

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const changelogStore = useChangelogStore();

const state = reactive<LocalState>({
  showSchemaDiffModal: false,
});

const latestChangelogSchema = ref("-- Loading latest changelog schema...");
const currentDatabaseSchema = ref("-- Loading current database schema...");

const updateDatabaseDrift = async () => {
  await databaseStore.updateDatabase(
    create(UpdateDatabaseRequestSchema, {
      database: create(DatabaseSchema$, {
        ...props.database,
        drifted: false,
      }),
      updateMask: { paths: ["drifted"] },
    })
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("database.drifted.new-baseline.successfully-established"),
  });
};

const fetchLatestChangelogSchema = async () => {
  try {
    const changelogs = await changelogStore.getOrFetchChangelogListOfDatabase(
      props.database.name,
      1, // Only get the latest one
      ChangelogView.FULL
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
      props.database.name
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
    if (show && props.database.drifted) {
      fetchLatestChangelogSchema();
      fetchCurrentDatabaseSchema();
    }
  }
);
</script>
