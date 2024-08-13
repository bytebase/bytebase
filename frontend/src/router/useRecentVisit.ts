import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { useDynamicLocalStorage } from "@/utils";

const STORAGE_KEY = "bb.space.recently_visited";
const MAX_HISTORY = 3;

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

  const record = (path: string) => {
    const list = recentVisit.value;
    const index = list.findIndex((item) => {
      // We treat the two URLs "the same" when their urls'
      //   `path` are the same (means ignoring querystring and hash).
      //   e.g. "/db?environment=5003" & "/db?environment=5005"
      // Because usually they are just different tab-panes
      //   or filters on the page.
      return getPath(item) === getPath(path);
    });
    if (index >= 0) {
      // current page exists in the history already
      // pull it out before next step
      list.splice(index, 1);
    }
    // then prepend the latest item to the queue
    list.unshift(path);

    // ensure the queue's length
    // should be no more than (MAX_HISTORY + 1)
    // because current page will always be the first one in the list
    // but it will be not shown in kbar
    while (list.length > MAX_HISTORY + 1) {
      list.pop();
    }
  };

  return {
    record,
    lastVisit,
  };
}

function getPath(url: string): string {
  return url.replace(/[?#].*$/, "");
}
