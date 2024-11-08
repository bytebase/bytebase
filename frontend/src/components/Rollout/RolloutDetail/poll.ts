import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { useRolloutStore } from "@/store";

export const usePollRollout = (rolloutName: string) => {
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
};
