import { useLocalStorage, useDebounce } from "@vueuse/core";
import { computed, ref, watch, nextTick } from "vue";
import { useRoute } from "vue-router";

type RecentVisit = {
  title: string;
  path: string;
};

const STORAGE_KEY = "bb.recently_visited";
const MAX_HISTORY = 3;

export function useRecentVisit() {
  const route = useRoute();
  const recentVisit = useLocalStorage(STORAGE_KEY, [] as RecentVisit[]);

  const lastVisit = computed(() => {
    if (recentVisit.value.length === 0) {
      return;
    }
    return recentVisit.value[0];
  });

  const currentRoute = computed(() => {
    if (route.matched.length === 0) {
      // When the application is landing, vue router will
      //   perform a redirection from "/" to current path
      //   e.g. if landing URL is "/instance", this redirection
      //   will be "/" -> "/instance".
      // This even happens when "/" -> "/".
      // `route.matched` is an empty array means this kind
      //   of redirection is going to be performed right now.
      // We don't want to record the first "/" as a visit
      //   because Home page is not shown at this time.
      return undefined;
    }
    return {
      name: route.name?.toString() ?? "",
      path: route.fullPath,
    };
  });
  const currentPage = ref<RecentVisit>();

  watch(
    // Debounce the listener so we can skip internal immediate redirections
    useDebounce(currentRoute, 50),
    async (currentRoute) => {
      if (!currentRoute) return;
      if (currentRoute.path.startsWith("/auth") || currentRoute.path.startsWith("/sql-editor")) {
        // ignore auth related pages
        // kbar is invisible on these pages
        // and navigating to these pages does not make sense
        return;
      }
      // wait for vue flushing
      // otherwise `document.title` will not be dynamically updated
      await nextTick();
      const { title } = document;
      currentPage.value = {
        title,
        ...currentRoute,
      };
    },
    { immediate: true }
  );

  watch(
    currentPage,
    (curr) => {
      if (!curr) return;
      const list = recentVisit.value;
      const index = list.findIndex((item) => {
        // We treat the two URLs "the same" when their urls'
        //   `path` are the same (means ignoring querystring and hash).
        //   e.g. "/db?environment=5003" & "/db?environment=5005"
        // Because usually they are just different tab-panes
        //   or filters on the page.
        return getPath(item.path) === getPath(item.path);
      });
      if (index >= 0) {
        // current page exists in the history already
        // pull it out before next step
        list.splice(index, 1);
      }
      // then prepend the latest item to the queue
      list.unshift(curr);

      // ensure the queue's length
      // should be no more than (MAX_HISTORY + 1)
      // because current page will always be the first one in the list
      // but it will be not shown in kbar
      while (list.length > MAX_HISTORY + 1) {
        list.pop();
      }
    },
    {
      immediate: true,
    }
  );

  return {
    recentVisit,
    lastVisit,
  };
}

function getPath(url: string): string {
  return url.replace(/[?#].*$/, "");
}
