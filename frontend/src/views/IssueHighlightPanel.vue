<template>
  <div class="md:flex md:items-center md:justify-between">
    <div class="flex-1 min-w-0">
      <div class="flex flex-col">
        <div class="flex items-center">
          <div>
            <IssueStatusIcon
              v-if="!create"
              :issueStatus="issue.status"
              :taskStatus="activeTask(issue.pipeline).status"
            />
          </div>
          <BBTextField
            class="ml-2 my-0.5 w-full text-lg font-bold"
            :disabled="!allowEdit"
            :required="true"
            :focusOnMount="create"
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
            <span href="#" class="font-medium text-control">{{
              moment(issue.updatedTs * 1000).format("LLL")
            }}</span>
          </p>
        </div>
      </div>
    </div>
    <div class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
      <slot />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, watch, PropType } from "vue";
import IssueStatusIcon from "../components/IssueStatusIcon.vue";
import { activeTask } from "../utils";
import { Issue } from "../types";

interface LocalState {
  editing: boolean;
  name: string;
}

export default {
  name: "IssueHighlightPanel",
  emits: ["update-name"],
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
  components: { IssueStatusIcon },
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      editing: false,
      name: props.issue.name,
    });

    watch(
      () => props.issue,
      (curIssue, _) => {
        state.name = curIssue.name;
      }
    );

    const trySaveName = (text: string) => {
      state.name = text;
      if (text != props.issue.name) {
        emit("update-name", state.name);
      }
    };

    return { state, activeTask, trySaveName };
  },
};
</script>
