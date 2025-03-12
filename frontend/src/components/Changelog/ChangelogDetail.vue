<template>
  <div class="focus:outline-none" tabindex="0" v-bind="$attrs">
    <NoPermissionPlaceholder v-if="!hasPermission" class="py-6" />
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
        <div class="flex-1 min-w-0 space-y-3">
          <!-- Summary -->
          <div class="flex items-center space-x-2">
            <ChangelogStatusIcon :status="changelog.status" />
            <NTag round>
              {{ changelog_TypeToJSON(changelog.type) }}
            </NTag>
            <NTag v-if="changelog.version" round>
              {{ $t("common.version") }} {{ changelog.version }}
            </NTag>
            <span class="text-xl">{{
              getDateForPbTimestamp(changelog.createTime)?.toLocaleString()
            }}</span>
          </div>
          <dl
            class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
          >
            <dt class="sr-only">{{ $t("common.issue") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.issue") }}&nbsp;-&nbsp;</span
              >
              <router-link
                :to="{
                  path: `/${changelog.issue}`,
                }"
                class="normal-link"
              >
                #{{ extractIssueUID(changelog.issue) }}
              </router-link>
            </dd>
          </dl>
        </div>
      </div>

      <div class="flex flex-col gap-y-6">
        <div v-if="affectedTables.length > 0">
          <span class="flex items-center text-lg text-main capitalize">
            {{ $t("changelog.affected-tables") }}
          </span>
          <div class="flex flex-wrap gap-x-3 gap-y-2">
            <span
              v-for="(affectedTable, i) in affectedTables"
              :key="`${i}.${affectedTable.schema}.${affectedTable.table}`"
              :class="[
                !affectedTable.dropped
                  ? 'text-blue-600 cursor-pointer'
                  : 'mb-2 text-gray-400 italic',
              ]"
            >
              {{ getAffectedTableDisplayName(affectedTable) }}
            </span>
          </div>
        </div>
        <div class="flex flex-col gap-y-2">
          <p class="flex items-center text-lg text-main capitalize">
            {{ $t("common.statement") }}
            <button
              tabindex="-1"
              class="btn-icon ml-1"
              @click.prevent="copyStatement"
            >
              <ClipboardIcon class="w-5 h-5" />
            </button>
          </p>
          <MonacoEditor
            class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
            :content="changelogStatement"
            :readonly="true"
            :auto-height="{ min: 120, max: 480 }"
          />
        </div>
        <div v-if="showSchemaSnapshot" class="flex flex-col gap-y-2">
          <p class="flex items-center text-lg text-main capitalize">
            Schema {{ $t("common.snapshot") }}
            <button
              tabindex="-1"
              class="btn-icon ml-1"
              @click.prevent="copySchema"
            >
              <ClipboardIcon class="w-5 h-5" />
            </button>
          </p>
          <div class="flex flex-row items-center gap-x-2">
            <div v-if="allowShowDiff" class="flex space-x-1 items-center">
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
            :original="changelog.prevSchema"
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
import { ClipboardIcon } from "lucide-vue-next";
import { NSwitch, NTag } from "naive-ui";
import { computed, reactive, watch, unref } from "vue";
import { BBSpin } from "@/bbkit";
import { DiffEditor, MonacoEditor } from "@/components/MonacoEditor";
import {
  pushNotification,
  useChangelogStore,
  useDBSchemaV1Store,
  useDatabaseV1ByName,
} from "@/store";
import { getDateForPbTimestamp } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { Changelog } from "@/types/proto/v1/database_service";
import {
  changelog_TypeToJSON,
  ChangelogView,
} from "@/types/proto/v1/database_service";
import {
  extractIssueUID,
  toClipboard,
  getStatementSize,
  hasProjectPermissionV2,
  getAffectedTableDisplayName,
} from "@/utils";
import {
  getAffectedTablesOfChangelog,
  getChangelogChangeType,
} from "@/utils/v1/changelog";
import NoPermissionPlaceholder from "../misc/NoPermissionPlaceholder.vue";
import ChangelogStatusIcon from "./ChangelogStatusIcon.vue";

interface LocalState {
  loading: boolean;
  showDiff: boolean;
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
});

const { database } = useDatabaseV1ByName(props.database);

const hasPermission = computed(() =>
  hasProjectPermissionV2(database.value.projectEntity, "bb.changelogs.get")
);

const changelogName = computed(() => {
  return `${props.database}/changelogs/${props.changelogId}`;
});

const changelog = computed((): Changelog | undefined => {
  return changelogStore.getChangelogByName(changelogName.value);
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
    getStatementSize(changelog.value.statement).lt(
      changelog.value.statementSize
    )
  ) {
    statement = `${statement}${statement.endsWith("\n") ? "" : "\n"}...`;
  }
  return statement;
});

const affectedTables = computed(() => {
  if (changelog.value === undefined) {
    return [];
  }
  return getAffectedTablesOfChangelog(changelog.value);
});

const showSchemaSnapshot = computed(() => {
  return database.value.instanceResource.engine !== Engine.RISINGWAVE;
});

// "Show diff" feature is enabled when current migration has changed the schema.
const allowShowDiff = computed((): boolean => {
  if (!changelog.value) {
    return false;
  }
  return getChangelogChangeType(changelog.value.type) === "DDL";
});

const copyStatement = async () => {
  if (!changelogStatement.value) {
    return false;
  }
  toClipboard(changelogStatement.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Statement copied to clipboard.`,
    });
  });
};

const copySchema = async () => {
  if (!changelogSchema.value) {
    return false;
  }
  toClipboard(changelogSchema.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Schema copied to clipboard.`,
    });
  });
};

watch(
  [database.value.name, changelogName],
  async ([_, name]) => {
    state.loading = true;
    await Promise.all([
      dbSchemaStore.getOrFetchDatabaseMetadata({
        database: database.value.name,
        skipCache: false,
      }),
      changelogStore.getOrFetchChangelogByName(
        unref(name),
        ChangelogView.CHANGELOG_VIEW_FULL
      ),
    ]);
    state.loading = false;
  },
  { immediate: true }
);
</script>
