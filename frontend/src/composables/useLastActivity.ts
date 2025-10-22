import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { useDynamicLocalStorage } from "@/utils";

export const useLastActivity = () => {
  const currentUser = useCurrentUserV1();
  const lastActivityTs = useDynamicLocalStorage<number>(
    computed(() => `bb.last-activity-ts.${currentUser.value.name}`),
    Date.now()
  );
  return { lastActivityTs };
};
