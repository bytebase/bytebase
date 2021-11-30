<template>
  <div class="md:flex md:items-center md:justify-between">
    <div class="flex-1 min-w-0">
      <div class="flex flex-col">
        <div class="flex items-center">
          <div>
            <IssueStatusIcon
              v-if="!create"
              :issue-status="issue.status"
              :task-status="activeTask(issue.pipeline).status"
            />
          </div>
          <BBTextField
            class="ml-2 my-0.5 w-full text-lg font-bold"
            :disabled="!allowEdit"
            :required="true"
            :focus-on-mount="create"
            :bordered="false"
            :value="state.name"
            :placeholder="'Issue name'"
            @end-editing="(text) => trySaveName(text)"
          />
        </div>
        <div v-if="!create">
          <p class="text-sm text-control-light">
            #{{ issue.id }} opened by
            <router-link
              :to="`/u/${issue.creator.id}`"
              class="font-medium text-control hover:underline"
              >{{ issue.creator.name }}</router-link
            >
            at
            {{ moment(issue.updatedTs * 1000).format("LLL") }}
          </p>
          <p
            v-if="pushEvent"
            class="
              mt-1
              text-sm text-control-light
              flex flex-row
              items-center
              space-x-1
            "
          >
            <template v-if="pushEvent.vcsType.startsWith('GITLAB')">
              <img class="h-4 w-auto" src="../assets/gitlab-logo.svg" />
            </template>
            <a :href="vcsBranchURL" target="_blank" class="normal-link">
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
          </p>
        </div>
      </div>
    </div>
    <div class="mt-4 flex space-x-3 md:mt-0 md:ml-4">
      <slot />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, watch, PropType, computed } from "vue";
import IssueStatusIcon from "../components/IssueStatusIcon.vue";
import { activeTask } from "../utils";
import { TaskDatabaseSchemaUpdatePayload, Issue, VCSPushEvent } from "../types";

interface LocalState {
  editing: boolean;
  name: string;
}

export default {
  name: "IssueHighlightPanel",
  components: { IssueStatusIcon },
  props: {
    issue: {
      required: true,
      type: Object as PropType<Issue>,
    },
    create: {
      required: true,
      type: Boolean,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  emits: ["update-name"],
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      editing: false,
      name: props.issue.name,
    });

    watch(
      () => props.issue,
      (curIssue) => {
        state.name = curIssue.name;
      }
    );

    const pushEvent = computed((): VCSPushEvent | undefined => {
      if (props.issue.type == "bb.issue.database.schema.update") {
        const payload = activeTask(props.issue.pipeline)
          .payload as TaskDatabaseSchemaUpdatePayload;
        return payload?.pushEvent;
      }
      return undefined;
    });

    const vcsType = computed((): string => {
      if (pushEvent.value) {
        if (pushEvent.value.vcsType.startsWith("GITLAB")) {
          return "GitLab";
        }
      }
      return "";
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

    const vcsBranchURL = computed((): string => {
      if (pushEvent.value) {
        if (pushEvent.value.vcsType == "GITLAB_SELF_HOST") {
          return `${pushEvent.value.repositoryURL}/-/tree/${vcsBranch.value}`;
        }
      }
      return "";
    });

    const trySaveName = (text: string) => {
      state.name = text;
      if (text != props.issue.name) {
        emit("update-name", state.name);
      }
    };

    return {
      state,
      activeTask,
      pushEvent,
      vcsType,
      vcsBranch,
      vcsBranchURL,
      trySaveName,
    };
  },
};
</script>
