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
      <div class="pb-6 border-b border-block-border">
        <div class="flex flex-col gap-y-4">
          <!-- Rollout Title -->
          <h2 v-if="changelog.planTitle && taskFullLink" class="text-2xl font-semibold">
            <router-link :to="taskFullLink" class="text-main hover:text-accent transition-colors">
              {{ changelog.planTitle }}
            </router-link>
          </h2>
          <h2 v-else-if="changelog.planTitle" class="text-2xl font-semibold text-main">
            {{ changelog.planTitle }}
          </h2>

          <!-- Metadata Row -->
          <div class="flex items-center gap-x-3 text-sm text-control-light">
            <div class="flex items-center gap-x-2">
              <ChangelogStatusIcon :status="changelog.status" />
              <span>{{ getChangelogChangeType(changelog.type) }}</span>
            </div>
            <span v-if="formattedCreateTime">•</span>
            <span v-if="formattedCreateTime">
              {{ formattedCreateTime }}
            </span>
          </div>
        </div>
      </div>

      <div class="flex flex-col gap-y-6">
        <div v-if="showSchemaSnapshot" class="flex flex-col gap-y-2">
          <p class="flex items-center text-lg text-main capitalize gap-x-2">
            Schema {{ $t("common.snapshot") }}
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
              v-if="allowRollback && state.showDiff"
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
import { NButton, NSwitch } from "naive-ui";
import { computed, reactive, unref, watch } from "vue";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { DiffEditor, MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import { PROJECT_V1_ROUTE_SYNC_SCHEMA } from "@/router/dashboard/projectV1";
import {
  useChangelogStore,
  useDatabaseV1ByName,
  useDBSchemaV1Store,
} from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Status,
  Changelog_Type,
  ChangelogView,
} from "@/types/proto-es/v1/database_service_pb";
import { extractProjectResourceName, wrapRefAsPromise } from "@/utils";
import { getChangelogChangeType } from "@/utils/v1/changelog";
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
const { allowAlterSchema } = useDatabaseDetailContext();

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
  return getDateForPbTimestampProtoEs(
    changelog.value.createTime
  )?.toLocaleString();
});

const changelogSchema = computed(() => {
  if (!changelog.value) {
    return "";
  }
  return changelog.value.schema;
});

const showSchemaSnapshot = computed(() => {
  return true;
});

// "Show diff" feature is enabled when current migration has changed the schema.
const allowShowDiff = computed((): boolean => {
  if (!changelog.value) {
    return false;
  }
  return changelog.value.type === Changelog_Type.MIGRATE;
});

// Get the previous changelog's schema as the "before" state for diff
const previousSchema = computed((): string => {
  if (!state.previousChangelog) {
    return "";
  }
  return state.previousChangelog.schema;
});

// Allow rollback for completed MIGRATE changelogs when user has alter schema permission
const allowRollback = computed((): boolean => {
  if (!changelog.value || !allowAlterSchema.value) {
    return false;
  }
  return (
    changelog.value.type === Changelog_Type.MIGRATE &&
    changelog.value.status === Changelog_Status.DONE
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
    }

    state.loading = false;
  },
  { immediate: true }
);
</script>
