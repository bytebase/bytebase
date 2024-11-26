import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useDynamicLocalStorage } from "@/utils";

const STORAGE_KEY = "bb.space.recently_visited";
const MAX_HISTORY = 10;

export function useRecentVisit() {
  const currentUser = useCurrentUserV1();

  const recentVisit = useDynamicLocalStorage<string[]>(
    computed(() => `${STORAGE_KEY}.${currentUser.value.name}`),
    []
  );

  const lastVisit = computed(() => {
    if (recentVisit.value.length === 0) {
      return;
    }
    return recentVisit.value[0];
  });

  const lastVisitProjectPath = computed(() => {
    for (const visit of recentVisit.value) {
      if (visit.startsWith(`/${projectNamePrefix}`)) {
        return visit;
      }
    }
    return "";
  });

  const remove = (path: string) => {
    const index = recentVisit.value.findIndex((item) => {
      // We treat the two URLs "the same" when their urls'
      //   `path` are the same (means ignoring querystring and hash).
      //   e.g. "/db?environment=5003" & "/db?environment=5005"
      // Because usually they are just different tab-panes
      //   or filters on the page.
      return getPath(item) === getPath(path);
    });
    if (index >= 0) {
      recentVisit.value.splice(index, 1);
    }
  };

  const record = (path: string) => {
    // current page exists in the history already
    // pull it out before next step
    remove(path);

    // ensure the queue's length
    // should be no more than (MAX_HISTORY + 1)
    // because current page will always be the first one in the list
    // but it will be not shown in kbar
    while (recentVisit.value.length > MAX_HISTORY + 1) {
      recentVisit.value.pop();
    }

    recentVisit.value.unshift(path);
  };

  return {
    remove,
    record,
    lastVisit,
    lastVisitProjectPath,
  };
}

function getPath(url: string): string {
  return url.replace(/[?#].*$/, "");
}
