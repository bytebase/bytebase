<template>
  <div class="focus:outline-hidden" tabindex="0" v-bind="$attrs">
    <div
      v-if="state.loading"
      class="flex items-center justify-center py-2 text-gray-400 text-sm"
    >
      <BBSpin />
    </div>
    <main v-else-if="changelog" class="flex flex-col relative gap-y-6">
      <!-- Highlight Panel -->
        <div class="flex flex-col gap-y-4">
          <!-- Plan Title -->
          <h2 v-if="changelog.planTitle" class="text-2xl font-semibold text-main">
            {{ changelog.planTitle }}
          </h2>

          <!-- Metadata Row -->
          <div class="flex items-center gap-x-3 text-sm text-control-light">
            <div class="flex items-center gap-x-2">
              <ChangelogStatusIcon :status="changelog.status" />
            </div>
            <span v-if="formattedCreateTime">•</span>
            <span v-if="formattedCreateTime">
              {{ formattedCreateTime }}
            </span>
          </div>
        </div>

      <div class="flex flex-col gap-y-6">
        <!-- Task Run Logs Section -->
        <div v-if="changelog.taskRun" class="flex flex-col gap-y-2">
          <div class="flex items-center justify-between">
            <p class="text-lg text-main">
              {{ $t("issue.task-run.logs") }}
            </p>
            <router-link
              v-if="taskFullLink"
              :to="taskFullLink"
              class="flex items-center gap-x-1 text-sm text-control-light hover:text-accent transition-colors"
            >
              {{ $t("common.show-more") }}
              <ArrowUpRightIcon class="w-4 h-4" />
            </router-link>
          </div>
          <TaskRunLogViewer
            v-if="changelog?.taskRun && showTaskRunLogs"
            :task-run-name="changelog.taskRun"
          />
          <div v-else class="text-sm text-control-light">
            {{ $t("common.no-data") }}
          </div>
        </div>

        <!-- Schema Snapshot Section -->
        <div v-if="showSchemaSnapshot" class="flex flex-col gap-y-2">
          <p class="flex items-center text-lg text-main capitalize gap-x-2">
            Schema {{ $t("common.snapshot") }}
            <span v-if="formattedSchemaSize" class="text-sm font-normal text-control-light">
              ({{ formattedSchemaSize }})
            </span>
            <CopyButton size="small" :content="changelogSchema" />
          </p>
          <div class="flex flex-row items-center justify-between gap-x-2">
            <div class="flex items-center gap-x-2">
              <div v-if="allowShowDiff" class="flex gap-x-1 items-center">
                <NSwitch
                  :value="state.showDiff"
                  size="small"
                  data-label="bb-changelog-diff-switch"
                  @update:value="state.showDiff = $event"
                />
                <span class="text-sm font-semibold">
                  {{ $t("changelog.show-diff") }}
                </span>
              </div>
              <div class="textinfolabel">
                {{ $t("changelog.schema-snapshot-after-change") }}
              </div>
              <div v-if="!allowShowDiff" class="text-sm font-normal text-accent">
                ({{ $t("changelog.no-schema-change") }})
              </div>
            </div>
            <NButton
              v-if="allowRollback"
              size="small"
              @click="handleRollback"
            >
              {{ $t("common.rollback") }}
            </NButton>
          </div>

          <DiffEditor
            v-if="state.showDiff"
            class="h-auto max-h-[600px] min-h-[120px] border rounded-md text-sm overflow-clip"
            :original="previousSchema"
            :modified="changelog.schema"
            :readonly="true"
            :auto-height="{ min: 120, max: 600 }"
          />
          <MonacoEditor
            v-else-if="changelog.schema"
            class="h-auto max-h-[600px] min-h-[120px] border rounded-md text-sm overflow-clip relative"
            :content="changelogSchema"
            :readonly="true"
            :auto-height="{ min: 120, max: 600 }"
          />
          <div v-else>
            {{ $t("changelog.current-schema-empty") }}
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts" setup>
import { ArrowUpRightIcon } from "lucide-vue-next";
import { NButton, NSwitch } from "naive-ui";
import { computed, reactive, unref, watch } from "vue";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { DiffEditor, MonacoEditor } from "@/components/MonacoEditor";
import { TaskRunLogViewer } from "@/components/RolloutV1/components/TaskRun/TaskRunLogViewer";
import { CopyButton } from "@/components/v2";
import { PROJECT_V1_ROUTE_SYNC_SCHEMA } from "@/router/dashboard/projectV1";
import {
  useChangelogStore,
  useDatabaseV1ByName,
  useDBSchemaV1Store,
} from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Status,
  ChangelogView,
} from "@/types/proto-es/v1/database_service_pb";
import {
  bytesToString,
  extractProjectResourceName,
  formatAbsoluteDateTime,
  getInstanceResource,
  wrapRefAsPromise,
} from "@/utils";
import { instanceV1SupportsSchemaRollback } from "@/utils/v1/instance";
import ChangelogStatusIcon from "./ChangelogStatusIcon.vue";

interface LocalState {
  loading: boolean;
  showDiff: boolean;
  previousChangelog?: Changelog;
}

const props = defineProps<{
  instance: string;
  database: string;
  changelogId: string;
}>();

const router = useRouter();
const dbSchemaStore = useDBSchemaV1Store();
const changelogStore = useChangelogStore();
const state = reactive<LocalState>({
  loading: false,
  showDiff: false,
  previousChangelog: undefined,
});

const { database, ready } = useDatabaseV1ByName(props.database);
const { allowAlterSchema, isDefaultProject } = useDatabaseDetailContext();

const changelogName = computed(() => {
  return `${props.database}/changelogs/${props.changelogId}`;
});

const changelog = computed((): Changelog | undefined => {
  return changelogStore.getChangelogByName(changelogName.value);
});

const taskFullLink = computed(() => {
  if (!changelog.value?.taskRun) {
    return "";
  }
  const parts = changelog.value.taskRun.split("/taskRuns/");
  if (parts.length !== 2) {
    return "";
  }
  return `/${parts[0]}`;
});

const formattedCreateTime = computed(() => {
  if (!changelog.value) {
    return "";
  }
  return formatAbsoluteDateTime(
    getTimeForPbTimestampProtoEs(changelog.value.createTime)
  );
});

const changelogSchema = computed(() => {
  if (!changelog.value) {
    return "";
  }
  return changelog.value.schema;
});

const formattedSchemaSize = computed(() => {
  if (!changelog.value || !changelog.value.schemaSize) {
    return "";
  }
  return bytesToString(Number(changelog.value.schemaSize));
});

const showSchemaSnapshot = computed(() => {
  return true;
});

// "Show diff" feature is enabled when current migration has changed the schema.
const allowShowDiff = computed((): boolean => {
  if (!changelog.value) {
    return false;
  }
  return true;
});

// Get the previous changelog's schema as the "before" state for diff
const previousSchema = computed((): string => {
  if (!state.previousChangelog) {
    return "";
  }
  return state.previousChangelog.schema;
});

// Allow rollback for completed MIGRATE changelogs when user has alter schema permission
// and the database engine supports schema diff rollback
const allowRollback = computed((): boolean => {
  if (isDefaultProject.value) {
    return false;
  }
  if (!changelog.value || !allowAlterSchema.value || !database.value) {
    return false;
  }
  // Check if engine supports schema diff rollback (GenerateMigration in backend)
  if (
    !instanceV1SupportsSchemaRollback(
      getInstanceResource(database.value).engine
    )
  ) {
    return false;
  }
  return changelog.value.status === Changelog_Status.DONE;
});

// Only show task run logs for completed or failed changelogs
const showTaskRunLogs = computed(() => {
  if (!changelog.value?.taskRun) return false;
  return (
    changelog.value.status === Changelog_Status.DONE ||
    changelog.value.status === Changelog_Status.FAILED
  );
});

const handleRollback = () => {
  if (!changelog.value || !database.value) {
    return;
  }

  router.push({
    name: PROJECT_V1_ROUTE_SYNC_SCHEMA,
    params: {
      projectId: extractProjectResourceName(database.value.project),
    },
    query: {
      changelog: changelog.value.name,
      target: database.value.name,
      rollback: "true",
    },
  });
};

watch(
  [database.value.name, changelogName],
  async () => {
    await wrapRefAsPromise(ready, true);
    state.loading = true;
    await Promise.all([
      dbSchemaStore.getOrFetchDatabaseMetadata({
        database: database.value.name,
        skipCache: false,
      }),
      changelogStore.getOrFetchChangelogByName(
        unref(changelogName),
        ChangelogView.FULL
      ),
    ]);

    // Fetch the previous changelog to get the "before" schema for diff
    if (changelog.value) {
      try {
        const prevChangelog = await changelogStore.fetchPreviousChangelog(
          unref(changelogName)
        );
        state.previousChangelog = prevChangelog;
      } catch (error) {
        console.error("Failed to fetch previous changelog:", error);
        state.previousChangelog = undefined;
      }

      // Show diff by default for all changelogs
      state.showDiff = true;
    }

    state.loading = false;
  },
  { immediate: true }
);
</script>
