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
            {{ migrationHistory.engine }} {{ migrationHistory.type }} -
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
        <div v-highlight class="border px-2 whitespace-pre-wrap w-full">
          {{ migrationHistory.statement }}
        </div>
        <code-diff
          v-if="showMigrationChange"
          el-id="schema"
          :title="'Schema ' + $t('common.snapshot')"
          :switcher-label="$t('migration-history.show-diff')"
          :info-switch-on-diff="$t('migration-history.left-vs-right')"
          :info-switch-off-diff="
            $t('migration-history.schema-snapshot-after-migration')
          "
          :info-no-diff="$t('migration-history.no-schema-change')"
          :old-code="migrationHistory.schemaPrev"
          :new-code="migrationHistory.schema"
        />
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, reactive, defineComponent } from "vue";
import { useStore } from "vuex";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import CodeDiff from "../components/CodeDiff.vue";
import MigrationHistoryStatusIcon from "../components/MigrationHistoryStatusIcon.vue";
import { idFromSlug, nanosecondsToString } from "../utils";
import {
  MigrationHistory,
  MigrationHistoryPayload,
  VCSPushEvent,
} from "../types";

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
    const store = useStore();

    const migrationHistory = computed((): MigrationHistory => {
      return store.getters["instance/migrationHistoryById"](
        idFromSlug(props.migrationHistorySlug)
      );
    });

    // Baseline migration should NOT cause schema change,
    // thus we only show migration change if there is unexpected schema change.
    //
    // In normal migration, always show the migration change.
    const showMigrationChange = computed(
      (): boolean =>
        migrationHistory.value.type !== "BASELINE" ||
        migrationHistory.value.schemaPrev !== migrationHistory.value.schema
    );

    const database = computed(() => {
      return store.getters["database/databaseById"](
        idFromSlug(props.databaseSlug)
      );
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
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: `Statement copied to clipboard.`,
        });
      });
    };

    return {
      nanosecondsToString,
      database,
      migrationHistory,
      showMigrationChange,
      pushEvent,
      vcsBranch,
      vcsBranchUrl,
      copyStatement,
    };
  },
});
</script>
