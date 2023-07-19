<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <main v-if="changeHistory" class="flex-1 relative pb-8 overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="px-4 pb-4 border-b border-block-border md:flex md:items-center md:justify-between"
      >
        <div class="flex-1 min-w-0 space-y-3">
          <!-- Summary -->
          <div class="pt-2 flex items-center space-x-2">
            <ChangeHistoryStatusIcon :status="changeHistory.status" />
            <h1 class="text-xl font-bold leading-6 text-main truncate">
              {{ $t("common.version") }} {{ changeHistory.version }}
            </h1>
          </div>
          <p class="text-control">
            {{ changeHistory_SourceToJSON(changeHistory.source) }}
            {{ changeHistory_TypeToJSON(changeHistory.type) }} -
            {{ changeHistory.description }}
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
                :to="`/issue/${extractIssueUID(changeHistory.issue)}`"
                class="normal-link"
              >
                {{ extractIssueUID(changeHistory.issue) }}
              </router-link>
            </dd>
            <dt class="sr-only">{{ $t("common.duration") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.duration") }}&nbsp;-&nbsp;</span
              >
              {{ humanizeDurationV1(changeHistory.executionDuration) }}
            </dd>
            <dt class="sr-only">{{ $t("common.creator") }}</dt>
            <dd v-if="creator" class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.creator") }}&nbsp;-&nbsp;</span
              >
              {{ creator.title }}
            </dd>
            <dt class="sr-only">{{ $t("common.created-at") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.created-at") }}&nbsp;-&nbsp;</span
              >
              {{ humanizeDate(changeHistory.createTime) }} by
              {{ `version ${changeHistory.releaseVersion}` }}
            </dd>
          </dl>
          <div
            v-if="pushEvent"
            class="mt-1 text-sm text-control-light flex flex-row items-center space-x-1"
          >
            <template
              v-if="vcsTypeToJSON(pushEvent?.vcsType).startsWith('GITLAB')"
            >
              <img class="h-4 w-auto" src="@/assets/gitlab-logo.svg" />
            </template>
            <template
              v-if="vcsTypeToJSON(pushEvent?.vcsType).startsWith('GITHUB')"
            >
              <img class="h-4 w-auto" src="@/assets/github-logo.svg" />
            </template>
            <template
              v-if="vcsTypeToJSON(pushEvent?.vcsType).startsWith('BITBUCKET')"
            >
              <img class="h-4 w-auto" src="@/assets/bitbucket-logo.svg" />
            </template>
            <a :href="vcsBranchUrl" target="_blank" class="normal-link">
              {{ `${vcsBranch}@${pushEvent.repositoryFullPath}` }}
            </a>
            <span v-if="vcsCommit">
              {{ $t("common.commit") }}
              <a :href="vcsCommit.url" target="_blank" class="normal-link">
                {{ vcsCommit.id.substring(0, 7) }}:
              </a>
              <span class="text-main mr-1">{{ vcsCommit.title }}</span>
              <i18n-t keypath="change-history.commit-info">
                <template #author>
                  {{ pushEvent.authorName }}
                </template>
                <template #time>
                  {{ humanizeDate(vcsCommit.createdTime) }}
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
          :code="changeHistory.statement"
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
          :old-string="changeHistory.prevSchema"
          :new-string="changeHistory.schema"
          output-format="side-by-side"
          data-label="bb-change-history-code-diff-block"
        />
        <template v-else>
          <highlight-code-block
            v-if="changeHistory.schema"
            class="border mt-2 px-2 whitespace-pre-wrap w-full"
            :code="changeHistory.schema"
            data-label="bb-change-history-code-block"
          />
          <div v-else class="mt-2">
            {{ $t("change-history.current-schema-empty") }}
          </div>
        </template>
      </div>
    </main>

    <BBModal
      v-if="changeHistory && previousHistory && state.viewDrift"
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
          :new-string="changeHistory.schema"
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

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { CodeDiff } from "v-code-diff";
import {
  changeHistoryLink,
  extractIssueUID,
  extractUserResourceName,
  uidFromSlug,
} from "@/utils";
import {
  pushNotification,
  useChangeHistoryStore,
  useDatabaseV1Store,
  useUserStore,
} from "@/store";
import {
  ChangeHistory,
  ChangeHistory_Type,
  changeHistory_SourceToJSON,
  changeHistory_TypeToJSON,
} from "@/types/proto/v1/database_service";
import { PushEvent, VcsType, vcsTypeToJSON } from "@/types/proto/v1/vcs";
import ChangeHistoryStatusIcon from "@/components/ChangeHistory/ChangeHistoryStatusIcon.vue";

interface LocalState {
  showDiff: boolean;
  viewDrift: boolean;
}

const props = defineProps<{
  instance: string;
  database: string;
  changeHistorySlug: string;
}>();

const changeHistoryStore = useChangeHistoryStore();

const changeHistoryParent = computed(() => {
  return `instances/${props.instance}/databases/${props.database}`;
});
const changeHistoryUID = computed(() => {
  return uidFromSlug(props.changeHistorySlug);
});
const changeHistoryName = computed(() => {
  return `${changeHistoryParent.value}/changeHistories/${changeHistoryUID.value}`;
});

watch(
  changeHistoryParent,
  (parent) => {
    useDatabaseV1Store().getOrFetchDatabaseByName(parent);
    changeHistoryStore.fetchChangeHistoryList({
      parent,
    });
  },
  { immediate: true }
);
watch(
  changeHistoryName,
  (name) => {
    changeHistoryStore.fetchChangeHistory({
      name,
    });
  },
  { immediate: true }
);

// get all change histories before (include) the one of given id, ordered by descending version.
const prevChangeHistoryList = computed(() => {
  const changeHistoryList = changeHistoryStore.changeHistoryListByDatabase(
    changeHistoryParent.value
  );

  // The returned change history list has been ordered by `id` DESC or (`namespace` ASC, `sequence` DESC) .
  // We can obtain prevChangeHistoryList by cutting up the array by the `changeHistoryId`.
  const idx = changeHistoryList.findIndex(
    (history) => history.uid === changeHistoryUID.value
  );
  if (idx === -1) {
    return [];
  }
  return changeHistoryList.slice(idx);
});

// changeHistory is the latest migration NOW.
const changeHistory = computed((): ChangeHistory | undefined => {
  if (prevChangeHistoryList.value.length > 0) {
    return prevChangeHistoryList.value[0];
  }
  return changeHistoryStore.getChangeHistoryByName(changeHistoryName.value)!;
});

// previousHistory is the last change history before the one of given id.
// Only referenced if hasDrift is true.
const previousHistory = computed((): ChangeHistory | undefined => {
  return prevChangeHistoryList.value[1];
});

// "Show diff" feature is enabled when current migration has changed the schema.
const allowShowDiff = computed((): boolean => {
  if (!changeHistory.value) {
    return false;
  }
  return changeHistory.value.schema !== changeHistory.value.prevSchema;
});

// A schema drift is detected when the schema AFTER previousHistory has been
// changed unexpectedly BEFORE current changeHistory.
const hasDrift = computed((): boolean => {
  if (!changeHistory.value) {
    return false;
  }
  if (changeHistory.value.type === ChangeHistory_Type.BASELINE) {
    return false;
  }

  return (
    prevChangeHistoryList.value.length > 1 && // no drift if no previous change history
    previousHistory.value?.schema !== changeHistory.value.prevSchema
  );
});

const creator = computed(() => {
  if (!changeHistory.value) {
    return undefined;
  }
  const email = extractUserResourceName(changeHistory.value.creator);
  return useUserStore().getUserByEmail(email);
});

const state = reactive<LocalState>({
  showDiff: allowShowDiff.value, // "Show diff" is turned on by default if available.
  viewDrift: false,
});

const previousHistoryLink = computed(() => {
  const previous = previousHistory.value;
  if (!previous) return "";
  return changeHistoryLink(previous);
});

const pushEvent = computed((): PushEvent | undefined => {
  return changeHistory.value?.pushEvent;
});

const vcsCommit = computed(() => {
  const fileCommit = pushEvent.value?.fileCommit;
  if (fileCommit && fileCommit.id) {
    return fileCommit;
  }
  const commit = pushEvent.value?.commits[0];
  if (commit && commit.id) {
    return commit;
  }
  return undefined;
});

const vcsBranch = computed((): string => {
  if (pushEvent.value) {
    if (
      pushEvent.value.vcsType === VcsType.GITLAB ||
      pushEvent.value.vcsType === VcsType.GITHUB ||
      pushEvent.value.vcsType === VcsType.BITBUCKET
    ) {
      const parts = pushEvent.value.ref.split("/");
      return parts[parts.length - 1];
    }
  }
  return "";
});

const vcsBranchUrl = computed((): string => {
  if (pushEvent.value) {
    if (pushEvent.value.vcsType === VcsType.GITLAB) {
      return `${pushEvent.value.repositoryUrl}/-/tree/${vcsBranch.value}`;
    } else if (pushEvent.value.vcsType === VcsType.GITHUB) {
      return `${pushEvent.value.repositoryUrl}/tree/${vcsBranch.value}`;
    } else if (pushEvent.value.vcsType === VcsType.BITBUCKET) {
      return `${pushEvent.value.repositoryUrl}/src/${vcsBranch.value}`;
    }
  }
  return "";
});

const copyStatement = () => {
  if (!changeHistory.value) {
    return false;
  }
  toClipboard(changeHistory.value.statement).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Statement copied to clipboard.`,
    });
  });
};

const copySchema = () => {
  if (!changeHistory.value) {
    return false;
  }
  toClipboard(changeHistory.value.schema).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Schema copied to clipboard.`,
    });
  });
};
</script>
