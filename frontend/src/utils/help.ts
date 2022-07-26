import { RouteLocationNormalized } from "vue-router";
import { useUIStateStore, useHelpStore } from "@/store";
import { RouteMapList } from "@/types";

let timer: number | null = null;
let RouteMapList: RouteMapList | null = null;

export const handleRouteChangedForHelp = async (
  to: RouteLocationNormalized
) => {
  const { name } = to;
  const uiStateStore = useUIStateStore();
  const helpStore = useHelpStore();

  if (!RouteMapList) {
    const res = await fetch("/help/routeMapList.json");
    RouteMapList = await res.json();
  }

  // Clear timer after every route change.
  if (timer) {
    clearTimeout(timer);
  }

  const helpName = RouteMapList?.find(
    (pair) => pair.routeName === name
  )?.helpName;

  if (helpName && !uiStateStore.getIntroStateByKey(`guide.${helpName}`)) {
    timer = window.setTimeout(() => {
      helpStore.showHelp(helpName, true);
      uiStateStore.saveIntroStateByKey({
        key: "environment.visit",
        newState: true,
      });
    }, 1000);
  }
};
