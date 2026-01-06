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
      <div
        class="pb-4 border-b border-block-border md:flex md:items-center md:justify-between"
      >
        <div class="flex-1 min-w-0 flex flex-col gap-y-3">
          <!-- Summary -->
          <div class="flex items-center gap-x-2">
            <ChangelogStatusIcon :status="changelog.status" />
            <NTag round>
              {{ getChangelogChangeType(changelog.type) }}
            </NTag>
            <NTag v-if="changelog.version" round>
              {{ $t("common.version") }} {{ changelog.version }}
            </NTag>
          </div>

          <dl v-if="taskFullLink" class="flex flex-col gap-y-1">
            <dt class="sr-only">{{ $t("common.task") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <router-link :to="taskFullLink" class="normal-link">
                {{ $t("changelog.task-at", { time: formattedCreateTime }) }}
              </router-link>
            </dd>
          </dl>
        </div>
      </div>

      <div class="flex flex-col gap-y-6">
        <div class="flex flex-col gap-y-2">
          <p class="flex items-center text-lg text-main capitalize gap-x-2">
            {{ $t("common.statement") }}
            <CopyButton size="small" :content="changelogStatement" />
          </p>
          <MonacoEditor
            class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
            :content="changelogStatement"
            :readonly="true"
            :auto-height="{ min: 120, max: 480 }"
          />
        </div>
        <div v-if="showSchemaSnapshot" class="flex flex-col gap-y-2">
          <p class="flex items-center text-lg text-main capitalize gap-x-2">
            Schema {{ $t("common.snapshot") }}
            <CopyButton size="small" :content="changelogSchema" />
          </p>
          <div class="flex flex-row items-center gap-x-2">
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
import { NSwitch, NTag } from "naive-ui";
import { computed, reactive, unref, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { DiffEditor, MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import {
  useChangelogStore,
  useDatabaseV1ByName,
  useDBSchemaV1Store,
} from "@/store";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Type,
  ChangelogView,
} from "@/types/proto-es/v1/database_service_pb";
import { getStatementSize, wrapRefAsPromise } from "@/utils";
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

const dbSchemaStore = useDBSchemaV1Store();
const changelogStore = useChangelogStore();
const state = reactive<LocalState>({
  loading: false,
  showDiff: false,
  previousChangelog: undefined,
});

const { database, ready } = useDatabaseV1ByName(props.database);

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

const changelogStatement = computed(() => {
  if (!changelog.value) {
    return "";
  }
  let statement = changelog.value.statement;
  if (
    getStatementSize(changelog.value.statement) < changelog.value.statementSize
  ) {
    statement = `${statement}${statement.endsWith("\n") ? "" : "\n"}...`;
  }
  return statement;
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
