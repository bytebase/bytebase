import { computed, onUnmounted, reactive, Ref, watch } from "vue";
import { useIssueStore } from "@/store";
import {
  Issue,
  IssueCreate,
  IssueType,
  NORMAL_POLL_INTERVAL,
  POLL_JITTER,
  POST_CHANGE_POLL_INTERVAL,
} from "@/types";
import { idFromSlug } from "@/utils";

type LocalPollState = {
  // Timer tracking the issue poller, we need this to cancel the outstanding one when needed.
  timer: ReturnType<typeof setTimeout> | undefined;
};

export const usePollIssue = (
  issueSlug: Ref<string>,
  issue: Ref<Issue | IssueCreate | undefined>
) => {
  const state = reactive<LocalPollState>({
    timer: undefined,
  });
  const create = computed(() => issueSlug.value.toLowerCase() == "new");
  const issueStore = useIssueStore();

  const stopPolling = () => {
    if (!state.timer) {
      return;
    }
    clearTimeout(state.timer);
  };

  // pollIssue invalidates the current timer and schedule a new timer in <<interval>> microseconds
  const pollIssue = (interval: number) => {
    stopPolling();

    const int = Math.max(
      1000,
      Math.min(interval, NORMAL_POLL_INTERVAL) +
        (Math.random() * 2 - 1) * POLL_JITTER
    );

    state.timer = setTimeout(() => {
      issueStore.fetchIssueById(idFromSlug(issueSlug.value));

      pollIssue(Math.min(interval * 2, NORMAL_POLL_INTERVAL));
    }, int);
  };

  const pollOnCreateStateChange = () => {
    if (!issue.value) {
      return;
    }

    let interval = NORMAL_POLL_INTERVAL;
    // We will poll faster if meets either of the condition
    // 1. Created the database create issue, expect creation result quickly.
    // 2. Update the database schema, will do connection and syntax check.
    const isNearNow =
      Date.now() - (issue.value as Issue).updatedTs * 1000 < 5000;
    const { type } = issue.value;
    if (FASTER_TYPES.includes(type) && isNearNow) {
      interval = POST_CHANGE_POLL_INTERVAL;
    }
    pollIssue(interval);
  };

  watch(
    create,
    () => {
      if (create.value) {
        stopPolling();
      } else {
        pollOnCreateStateChange();
      }
    },
    { immediate: true }
  );

  onUnmounted(stopPolling);

  return pollIssue;
};

const FASTER_TYPES: IssueType[] = [
  "bb.issue.database.create",
  "bb.issue.database.schema.update",
  "bb.issue.database.data.update",
  "bb.issue.database.schema.update.ghost",
];
