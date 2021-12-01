<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <main class="flex-1 relative pb-8 overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="
          px-4
          pb-4
          border-b border-block-border
          md:flex md:items-center md:justify-between
        "
      >
        <div class="flex-1 min-w-0 space-y-3">
          <!-- Summary -->
          <div class="pt-2 flex items-center space-x-2">
            <MigrationHistoryStatusIcon :status="migrationHistory.status" />
            <h1 class="text-xl font-bold leading-6 text-main truncate">
              Version {{ migrationHistory.version }}
            </h1>
          </div>
          <p class="text-control">
            {{ migrationHistory.engine }} {{ migrationHistory.type }} -
            {{ migrationHistory.description }}
          </p>
          <dl
            class="
              flex flex-col
              space-y-1
              md:space-y-0 md:flex-row md:flex-wrap
            "
          >
            <dt class="sr-only">Issue</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Issue&nbsp;-&nbsp;</span>
              <router-link
                :to="`/issue/${migrationHistory.issueId}`"
                class="normal-link"
              >
                {{ migrationHistory.issueId }}
              </router-link>
            </dd>
            <dt class="sr-only">Duration</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Duration&nbsp;-&nbsp;</span>
              {{ secondsToString(migrationHistory.executionDuration) }}
            </dd>
            <dt class="sr-only">Creator</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Creator&nbsp;-&nbsp;</span>
              {{ migrationHistory.creator }}
            </dd>
            <dt class="sr-only">Created</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel">Created&nbsp;-&nbsp;</span>
              {{ humanizeTs(migrationHistory.createdTs) }} by
              {{ `version ${migrationHistory.releaseVersion}` }}
            </dd>
          </dl>
          <div
            v-if="pushEvent"
            class="
              mt-1
              text-sm text-control-light
              flex flex-row
              items-center
              space-x-1
            "
          >
            <template v-if="pushEvent?.vcsType.startsWith('GITLAB')">
              <img class="h-4 w-auto" src="../assets/gitlab-logo.svg" />
            </template>
            <a :href="vcsBranchUrl" target="_blank" class="normal-link">
              {{ `${vcsBranch}@${pushEvent.repositoryFullPath}` }}
            </a>
            <span>
              commit
              <a
                :href="pushEvent.fileCommit.url"
                target="_blank"
                class="normal-link"
              >
                {{ pushEvent.fileCommit.id.substring(0, 7) }}:
              </a>
              <span class="text-main">{{ pushEvent.fileCommit.title }}</span>
              by
              {{ pushEvent.authorName }} at
              {{ moment(pushEvent.fileCommit.createdTs * 1000).format("LLL") }}
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
          Statement
          <button
            tabindex="-1"
            class="btn-icon ml-1"
            @click.prevent="copyStatement"
          >
            <svg
              class="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
              ></path>
            </svg>
          </button>
        </a>
        <div v-highlight class="border px-2 whitespace-pre-wrap w-full">
          {{ migrationHistory.statement }}
        </div>
        <a
          id="schema"
          href="#schema"
          class="flex items-center text-lg text-main mt-6 hover:underline"
        >
          Schema Snapshot
          <button
            tabindex="-1"
            class="btn-icon ml-1"
            @click.prevent="copySchema"
          >
            <svg
              class="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
              ></path>
            </svg>
          </button>
        </a>
        <div class="flex flex-row items-center space-x-2 mt-2">
          <BBSwitch
            v-if="migrationHistory.schemaPrev != migrationHistory.schema"
            :label="'Show diff'"
            :value="state.showDiff"
            @toggle="
              (on) => {
                state.showDiff = on;
              }
            "
          />
          <div class="textinfolabel">
            {{
              state.showDiff
                ? "Prev (left) vs This version (right)"
                : "The schema snapshot recorded after applying this migration version"
            }}
          </div>
          <div
            v-if="migrationHistory.schemaPrev == migrationHistory.schema"
            class="text-sm font-normal text-accent"
          >
            (this migration has no schema change)
          </div>
        </div>
        <code-diff
          v-if="state.showDiff"
          class="mt-4 w-full"
          :old-string="migrationHistory.schemaPrev"
          :new-string="migrationHistory.schema"
          output-format="side-by-side"
        />
        <div
          v-else
          v-highlight
          class="border mt-2 px-2 whitespace-pre-wrap w-full"
        >
          {{ migrationHistory.schema }}
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { CodeDiff } from "v-code-diff";
import MigrationHistoryStatusIcon from "../components/MigrationHistoryStatusIcon.vue";
import { idFromSlug, secondsToString } from "../utils";
import {
  MigrationHistory,
  MigrationHistoryPayload,
  VCSPushEvent,
} from "../types";

interface LocalState {
  showDiff: boolean;
}

export default {
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

    const state = reactive<LocalState>({
      showDiff:
        migrationHistory.value.schema != migrationHistory.value.schemaPrev,
    });

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

    const copySchema = () => {
      toClipboard(migrationHistory.value.schema).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: `Schema copied to clipboard.`,
        });
      });
    };

    return {
      state,
      secondsToString,
      database,
      migrationHistory,
      pushEvent,
      vcsBranch,
      vcsBranchUrl,
      copyStatement,
      copySchema,
    };
  },
};
</script>
