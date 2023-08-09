import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useStorage, useDebounce } from "@vueuse/core";
import { computed, ref, watch, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

type RecentVisit = {
  title: string;
  url: string;
};

const STORAGE_KEY = "ui.kbar.recently_visited";
const MAX_HISTORY = 3;

export function useRecentVisit() {
  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const recentVisit = useStorage(STORAGE_KEY, [] as RecentVisit[]);
  const url = computed(() => {
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
    return route.fullPath;
  });
  const currentPage = ref<RecentVisit>();

  watch(
    // Debounce the listener so we can skip internal immediate redirections
    useDebounce(url, 50),
    async (url) => {
      if (!url) return;
      if (url.startsWith("/auth")) {
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
        url,
        title,
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
        //   e.g. "/environment#5001" & "/environment#5005"
        // Because usually they are just different tab-panes
        //   or filters on the page.
        return getPath(item.url) === getPath(curr.url);
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

  const actions = computed(() => {
    return recentVisit.value
      .slice(1) // The first item is current page, just skip it.
      .filter(({ title, url }) => title && url)
      .map(({ title, url }, index) =>
        defineAction({
          // here `id` looks like "bb.recent_visited.1"
          id: `bb.recently_visited.${index + 1}`,
          section: t("kbar.recently-visited"),
          name: title,
          subtitle: url,
          shortcut: ["g", `${index + 1}`],
          keywords: "recently visited",
          perform: () => router.push({ path: url }),
        })
      );
  });

  // prepend recent visit actions to kbar
  useRegisterActions(actions, true);
}

function getPath(url: string): string {
  return url.replace(/[?#].*$/, "");
}
