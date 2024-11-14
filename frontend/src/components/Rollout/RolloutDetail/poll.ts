import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { useRolloutStore } from "@/store";
import { type EventsEmmiter } from "./context";

export const usePollRollout = (rolloutName: string, emmiter: EventsEmmiter) => {
  const rolloutStore = useRolloutStore();

  const refreshRollout = async () => {
    await rolloutStore.fetchRolloutByName(rolloutName);
  };

  const poller = useProgressivePoll(refreshRollout, {
    interval: {
      min: 500,
      max: 10000,
      growth: 2,
      jitter: 500,
    },
  });

  poller.start();

  // When any task status action is triggered, we need to refresh the rollout.
  emmiter.on("task-status-action", () => {
    refreshRollout();
    poller.restart();
  });
};
