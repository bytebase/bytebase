import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { storageKeyLastActivity, useDynamicLocalStorage } from "@/utils";

export const useLastActivity = () => {
  const currentUser = useCurrentUserV1();
  const lastActivityTs = useDynamicLocalStorage<number>(
    computed(() => storageKeyLastActivity(currentUser.value.email)),
    Date.now()
  );
  return { lastActivityTs };
};
