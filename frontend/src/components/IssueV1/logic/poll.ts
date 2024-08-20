import { computed, watch } from "vue";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { experimentalFetchIssueByUID } from "@/store";
import {
  extractIssueUID,
  extractProjectResourceName,
  isValidTaskName,
} from "@/utils";
import { useIssueContext } from "./context";

export const usePollIssue = () => {
  const { isCreating, ready, issue, events, activeTask } = useIssueContext();

  const shouldPollIssue = computed(() => {
    return !isCreating.value && ready.value;
  });

  const refreshIssue = () => {
    if (!shouldPollIssue.value) return;

    experimentalFetchIssueByUID(
      extractIssueUID(issue.value.name),
      extractProjectResourceName(issue.value.project)
    ).then((updatedIssue) => (issue.value = updatedIssue));
  };

  const poller = useProgressivePoll(refreshIssue, {
    interval: {
      min: 500,
      max: 10000,
      growth: 2,
      jitter: 500,
    },
  });

  watch(
    shouldPollIssue,
    () => {
      if (shouldPollIssue.value) {
        poller.start();
      } else {
        poller.stop();
      }
    },
    {
      immediate: true,
    }
  );

  events.on("status-changed", ({ eager }) => {
    if (eager) {
      refreshIssue();
      poller.restart();
    }
  });

  watch(activeTask, (curr, prev) => {
    if (!isValidTaskName(curr.name)) return;
    if (curr.name !== prev.name) {
      events.emit("select-task", { task: curr });
    }
  });
};
