<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <main class="flex-1 relative pb-8 overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="px-4 pb-4 border-b border-block-border md:flex md:items-center md:justify-between"
      >
        <div class="flex-1 min-w-0 space-y-3">
          <!-- Summary -->
          <div class="pt-2 flex items-center space-x-2">
            <MigrationHistoryStatusIcon :status="migrationHistory.status" />
            <h1 class="text-xl font-bold leading-6 text-main truncate">
              {{ $t("common.version") }} {{ migrationHistory.version }}
            </h1>
          </div>
          <p class="text-control">
            {{ migrationHistory.source }} {{ migrationHistory.type }} -
            {{ migrationHistory.description }}
          </p>
          <dl
            class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
          >
            <dt class="sr-only">{{ $t("common.issue") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.issue") }}&nbsp;-&nbsp;</span
              >
              <router-link
                :to="`/issue/${migrationHistory.issueId}`"
                class="normal-link"
              >
                {{ migrationHistory.issueId }}
              </router-link>
            </dd>
            <dt class="sr-only">{{ $t("common.duration") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.duration") }}&nbsp;-&nbsp;</span
              >
              {{ nanosecondsToString(migrationHistory.executionDurationNs) }}
            </dd>
            <dt class="sr-only">{{ $t("common.creator") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.creator") }}&nbsp;-&nbsp;</span
              >
              {{ migrationHistory.creator }}
            </dd>
            <dt class="sr-only">{{ $t("common.created-at") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.created-at") }}&nbsp;-&nbsp;</span
              >
              {{ humanizeTs(migrationHistory.createdTs) }} by
              {{ `version ${migrationHistory.releaseVersion}` }}
            </dd>
          </dl>
          <div
            v-if="pushEvent"
            class="mt-1 text-sm text-control-light flex flex-row items-center space-x-1"
          >
            <template v-if="pushEvent?.vcsType.startsWith('GITLAB')">
              <img class="h-4 w-auto" src="../assets/gitlab-logo.svg" />
            </template>
            <a :href="vcsBranchUrl" target="_blank" class="normal-link">
              {{ `${vcsBranch}@${pushEvent.repositoryFullPath}` }}
            </a>
            <span>
              {{ $t("common.commit") }}
              <a
                :href="pushEvent.fileCommit.url"
                target="_blank"
                class="normal-link"
              >
                {{ pushEvent.fileCommit.id.substring(0, 7) }}:
              </a>
              <span class="text-main">{{ pushEvent.fileCommit.title }}</span>
              <i18n-t keypath="migration-history.commit-info">
                <template #author>
                  {{ pushEvent.authorName }}
                </template>
                <template #time>
                  {{
                    dayjs(pushEvent.fileCommit.createdTs * 1000).format("LLL")
                  }}
                </template>
              </i18n-t>
            </span>
          </div>
        </div>
      </div>

      <div class="mt-6 px-4">
        <a
          id="statement"
          href="#statement"
          class="flex items-center text-lg text-main mb-2 hover:underline"
        >
          {{ $t("common.statement") }}
          <button
            tabindex="-1"
            class="btn-icon ml-1"
            @click.prevent="copyStatement"
          >
            <heroicons-outline:clipboard class="w-6 h-6" />
          </button>
        </a>
        <code-highlight
          class="border px-2 whitespace-pre-wrap w-full"
          :code="migrationHistory.statement"
        />
        <a
          id="schema"
          href="#schema"
          class="flex items-center text-lg text-main mt-6 hover:underline capitalize"
        >
          Schema {{ $t("common.snapshot") }}
          <button
            tabindex="-1"
            class="btn-icon ml-1"
            @click.prevent="copySchema"
          >
            <heroicons-outline:clipboard class="w-6 h-6" />
          </button>
        </a>

        <div v-if="hasDrift" class="flex flex-row items-center space-x-2 mt-2">
          <div class="text-sm font-normal text-accent">
            ({{ $t("migration-history.schema-drift") }})
          </div>
          <span class="textinfolabel">
            {{ $t("migration-history.before-left-schema-choice") }}
          </span>
          <div>
            <BBSelect
              :selected-item="state.leftSelected"
              :item-list="['previousHistorySchema', 'currentHistorySchemaPrev']"
              @select-item="
                (value) => {
                  state.leftSelected = value;
                  state.leftSchema =
                    state.leftSelected === 'previousHistorySchema'
                      ? previousHistorySchema
                      : migrationHistory.schemaPrev;
                  state.showDiff = state.leftSchema !== state.rightSchema;
                }
              "
            >
              <template #menuItem="{ item: value }">
                {{
                  value === "previousHistorySchema"
                    ? $t(
                        "migration-history.left-schema-choice-prev-history-schema"
                      )
                    : $t(
                        "migration-history.left-schema-choice-current-history-schema-prev"
                      )
                }}
              </template>
            </BBSelect>
          </div>
          <span class="textinfolabel">
            {{ $t("migration-history.after-left-schema-choice") }}
          </span>
        </div>

        <div class="flex flex-row items-center space-x-2 mt-2">
          <BBSwitch
            v-if="state.leftSchema !== state.rightSchema"
            :label="$t('migration-history.show-diff')"
            :value="state.showDiff"
            @toggle="
              (on: any) => {
                state.showDiff = on;
              }
            "
          />
          <div class="textinfolabel">
            {{
              state.showDiff
                ? $t("migration-history.left-vs-right")
                : $t("migration-history.schema-snapshot-after-migration")
            }}
          </div>
          <div
            v-if="state.leftSchema === state.rightSchema"
            class="text-sm font-normal text-accent"
          >
            ({{ $t("migration-history.no-schema-change") }})
          </div>
        </div>
        <code-diff
          v-if="state.showDiff"
          class="mt-4 w-full"
          :old-string="state.leftSchema"
          :new-string="state.rightSchema"
          output-format="side-by-side"
        />
        <code-highlight
          v-else
          class="border mt-2 px-2 whitespace-pre-wrap w-full"
          :code="migrationHistory.schema"
        />
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, reactive, defineComponent } from "vue";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { CodeDiff } from "v-code-diff";
import MigrationHistoryStatusIcon from "../components/MigrationHistoryStatusIcon.vue";
import { idFromSlug, nanosecondsToString } from "../utils";
import {
  MigrationHistory,
  MigrationHistoryPayload,
  VCSPushEvent,
} from "../types";
import { BBSelect } from "../bbkit";
import { pushNotification, useDatabaseStore, useInstanceStore } from "@/store";

type LeftSchemaSelected =
  | "previousHistorySchema" // schema after last migration
  | "currentHistorySchemaPrev"; // schema before this migration

interface LocalState {
  showDiff: boolean;
  leftSelected: LeftSchemaSelected;
  // leftSchema is the schema snapshot at the left side of the diff.
  // Default to migrationHistory.schemaPrev. If drift is detected, it can be selected to be lastRecordedSchema.
  leftSchema: string;
  // rightSchema is the schema snapshot at the right side of the diff. Always migrationHistory.schema.
  rightSchema: string;
}

export default defineComponent({
  name: "MigrationHistoryDetail",
  components: { CodeDiff, MigrationHistoryStatusIcon, BBSelect },
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
    migrationHistorySlug: {
      required: true,
      type: String,
    },
  },
  setup(props) {
    const instanceStore = useInstanceStore();

    const database = computed(() => {
      return useDatabaseStore().getDatabaseById(idFromSlug(props.databaseSlug));
    });

    const migrationHistoryId = idFromSlug(props.migrationHistorySlug);

    // get all migration histories before (include) the one of given id, ordered by descending version.
    const prevMigrationHistoryList = computed((): MigrationHistory[] => {
      const migrationHistoryList =
        instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
          database.value.instance.id,
          database.value.name
        );

      // If migrationHistoryList does not contain current migration, it indicates cache stale.
      // Dispatch a fetch. When new data is returned, it will update computed value.
      if (migrationHistoryList.every((mh) => mh.id !== migrationHistoryId)) {
        instanceStore.fetchInstanceList();
      }

      return migrationHistoryList.filter((mh) => mh.id <= migrationHistoryId);
    });

    const migrationHistory = computed((): MigrationHistory => {
      if (prevMigrationHistoryList.value.length > 0) {
        return prevMigrationHistoryList.value[0];
      }
      return instanceStore.getMigrationHistoryById(migrationHistoryId)!;
    });

    // previousHistorySchema is the schema snapshot of the last migration history before the one of given id.
    // Only referenced if hasDrift is true.
    const previousHistorySchema = computed(
      (): string => prevMigrationHistoryList.value[1].schema
    );

    const hasDrift = computed(
      (): boolean =>
        prevMigrationHistoryList.value.length > 1 && // no drift if no previous migration history
        previousHistorySchema.value !== migrationHistory.value.schemaPrev
    );

    const state = reactive<LocalState>({
      showDiff:
        migrationHistory.value.schema != migrationHistory.value.schemaPrev,
      leftSelected: "currentHistorySchemaPrev",
      leftSchema: migrationHistory.value.schemaPrev,
      rightSchema: migrationHistory.value.schema,
    });

    const pushEvent = computed((): VCSPushEvent | undefined => {
      return (migrationHistory.value.payload as MigrationHistoryPayload)
        ?.pushEvent;
    });

    const vcsBranch = computed((): string => {
      if (pushEvent.value) {
        if (pushEvent.value.vcsType == "GITLAB_SELF_HOST") {
          const parts = pushEvent.value.ref.split("/");
          return parts[parts.length - 1];
        }
      }
      return "";
    });

    const vcsBranchUrl = computed((): string => {
      if (pushEvent.value) {
        if (pushEvent.value.vcsType == "GITLAB_SELF_HOST") {
          return `${pushEvent.value.repositoryUrl}/-/tree/${vcsBranch.value}`;
        }
      }
      return "";
    });

    const copyStatement = () => {
      toClipboard(migrationHistory.value.statement).then(() => {
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: `Statement copied to clipboard.`,
        });
      });
    };

    const copySchema = () => {
      toClipboard(migrationHistory.value.schema).then(() => {
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: `Schema copied to clipboard.`,
        });
      });
    };

    return {
      state,
      nanosecondsToString,
      database,
      migrationHistory,
      previousHistorySchema,
      hasDrift,
      pushEvent,
      vcsBranch,
      vcsBranchUrl,
      copyStatement,
      copySchema,
    };
  },
});
</script>
