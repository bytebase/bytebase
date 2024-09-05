import { watch } from "vue";
import { databaseForTask } from "@/components/IssueV1/logic";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { experimentalFetchIssueByUID, useChangeHistoryStore } from "@/store";
import type { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  extractIssueUID,
  extractProjectResourceName,
  isValidTaskName,
} from "@/utils";
import { flattenTaskV1List } from "@/utils";
import { useIssueContext } from "./context";

const clearChangeHistory = (issue: ComposedIssue) => {
  const changeHistoryStore = useChangeHistoryStore();
  const tasks = flattenTaskV1List(issue.rolloutEntity);
  for (const task of tasks) {
    const database = databaseForTask(issue, task);
    changeHistoryStore.clearCache(database.name);
  }
};

export const usePollIssue = () => {
  const { isCreating, ready, issue, events, activeTask } = useIssueContext();

  const refreshIssue = () => {
    if (isCreating.value || !ready.value) return;
    experimentalFetchIssueByUID(
      extractIssueUID(issue.value.name),
      extractProjectResourceName(issue.value.project)
    ).then((updatedIssue) => {
      if (
        issue.value.status !== IssueStatus.DONE &&
        updatedIssue.status === IssueStatus.DONE
      ) {
        clearChangeHistory(updatedIssue);
      }

      issue.value = updatedIssue;
    });
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
    () => [isCreating.value, ready.value],
    () => {
      if (!isCreating.value && ready.value) {
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
