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
              <i18n-t keypath="change-history.commit-info">
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
        <highlight-code-block
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

        <div v-if="hasDrift" class="flex items-center space-x-2 mt-2">
          <div class="flex items-center text-sm font-normal">
            <heroicons-outline:exclamation-circle
              class="w-5 h-5 mr-0.5 text-error"
            />
            <span>{{ $t("change-history.schema-drift-detected") }}</span>
          </div>
          <div
            class="normal-link text-sm"
            data-label="bb-change-history-view-drift-button"
            @click="state.viewDrift = true"
          >
            {{ $t("change-history.view-drift") }}
          </div>
        </div>

        <div class="flex flex-row items-center space-x-2 mt-2">
          <BBSwitch
            v-if="allowShowDiff"
            :label="$t('change-history.show-diff')"
            :value="state.showDiff"
            data-label="bb-change-history-diff-switch"
            @toggle="state.showDiff = $event"
          />
          <div class="textinfolabel">
            <i18n-t
              v-if="state.showDiff"
              tag="span"
              keypath="change-history.left-vs-right"
            >
              <template #prevLink>
                <router-link
                  v-if="previousHistory"
                  class="normal-link"
                  :to="previousHistoryLink"
                >
                  ({{ previousHistory.version }})
                </router-link>
              </template>
            </i18n-t>
            <template v-else>
              {{ $t("change-history.schema-snapshot-after-change") }}
            </template>
          </div>
          <div v-if="!allowShowDiff" class="text-sm font-normal text-accent">
            ({{ $t("change-history.no-schema-change") }})
          </div>
        </div>

        <code-diff
          v-if="state.showDiff"
          class="mt-4 w-full"
          :old-string="migrationHistory.schemaPrev"
          :new-string="migrationHistory.schema"
          output-format="side-by-side"
          data-label="bb-change-history-code-diff-block"
        />
        <template v-else>
          <highlight-code-block
            v-if="migrationHistory.schema"
            class="border mt-2 px-2 whitespace-pre-wrap w-full"
            :code="migrationHistory.schema"
            data-label="bb-change-history-code-block"
          />
          <div v-else class="mt-2">
            {{ $t("change-history.current-schema-empty") }}
          </div>
        </template>
      </div>
    </main>

    <BBModal
      v-if="previousHistory && state.viewDrift"
      @close="state.viewDrift = false"
    >
      <template #title>
        <span>{{ $t("change-history.schema-drift") }}</span>
        <span class="mx-2">-</span>
        <i18n-t tag="span" keypath="change-history.left-vs-right">
          <template #prevLink>
            <router-link class="normal-link" :to="previousHistoryLink">
              ({{ previousHistory.version }})
            </router-link>
          </template>
        </i18n-t>
      </template>

      <div class="space-y-4">
        <code-diff
          class="w-full"
          :old-string="previousHistory.schema"
          :new-string="migrationHistory.schema"
          output-format="side-by-side"
        />
        <div class="flex justify-end px-4">
          <button
            type="button"
            class="btn-primary"
            @click.prevent="state.viewDrift = false"
          >
            {{ $t("common.close") }}
          </button>
        </div>
      </div>
    </BBModal>
  </div>
</template>

<script lang="ts">
import { computed, reactive, defineComponent, onMounted } from "vue";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { CodeDiff } from "v-code-diff";
import MigrationHistoryStatusIcon from "../components/MigrationHistoryStatusIcon.vue";
import {
  idFromSlug,
  nanosecondsToString,
  migrationHistorySlug,
  migrationHistoryIdFromSlug,
} from "../utils";
import {
  MigrationHistory,
  MigrationHistoryPayload,
  VCSPushEvent,
} from "../types";
import { pushNotification, useDatabaseStore, useInstanceStore } from "@/store";

interface LocalState {
  showDiff: boolean;
  viewDrift: boolean;
}

export default defineComponent({
  name: "MigrationHistoryDetail",
  components: { CodeDiff, MigrationHistoryStatusIcon },
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

    const migrationHistoryId = migrationHistoryIdFromSlug(
      props.migrationHistorySlug
    );

    onMounted(() => {
      instanceStore.fetchMigrationHistory({
        instanceId: database.value.instance.id,
        databaseName: database.value.name,
      });
    });

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

      // The returned migration history list has been ordered by `sequence` DESC.
      // We can obtain prevMigrationHistoryList by cutting up the array by the `migrationHistoryId`.
      let met = false;
      return migrationHistoryList.filter((mh) => {
        if (mh.id === migrationHistoryId) {
          met = true;
        }
        return met;
      });
    });

    // migrationHistory is the latest migration NOW.
    const migrationHistory = computed((): MigrationHistory => {
      if (prevMigrationHistoryList.value.length > 0) {
        return prevMigrationHistoryList.value[0];
      }
      return instanceStore.getMigrationHistoryById(migrationHistoryId)!;
    });

    // previousHistory is the last migration history before the one of given id.
    // Only referenced if hasDrift is true.
    const previousHistory = computed((): MigrationHistory | undefined => {
      return prevMigrationHistoryList.value[1];
    });

    // "Show diff" feature is enabled when current migration has changed the schema.
    const allowShowDiff = computed((): boolean => {
      return (
        migrationHistory.value.schema !== migrationHistory.value.schemaPrev
      );
    });

    // A schema drift is detected when the schema AFTER previousHistory has been
    // changed unexpectedly BEFORE current migrationHistory.
    const hasDrift = computed((): boolean => {
      if (migrationHistory.value.type === "BASELINE") {
        return false;
      }

      return (
        prevMigrationHistoryList.value.length > 1 && // no drift if no previous migration history
        previousHistory.value?.schema !== migrationHistory.value.schemaPrev
      );
    });

    const state = reactive<LocalState>({
      showDiff: allowShowDiff.value, // "Show diff" is turned on by default if available.
      viewDrift: false,
    });

    const previousHistoryLink = computed(() => {
      const previous = previousHistory.value;
      if (!previous) return "";
      return `/db/${props.databaseSlug}/history/${migrationHistorySlug(
        previous.id,
        previous.version
      )}`;
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
        } else if (pushEvent.value.vcsType == "GITHUB_COM") {
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
        } else if (pushEvent.value.vcsType == "GITHUB_COM") {
          return `${pushEvent.value.repositoryUrl}/tree/${vcsBranch.value}`;
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
      previousHistory,
      allowShowDiff,
      hasDrift,
      previousHistoryLink,
      pushEvent,
      vcsBranch,
      vcsBranchUrl,
      copyStatement,
      copySchema,
    };
  },
});
</script>
