import { RouteLocationNormalized } from "vue-router";
import { useUIStateStore, useHelpStore } from "@/store";
import { RouteMap } from "@/types";

let timer: number | null = null;
let routeMap: RouteMap | null = null;

export const handleRouteChangedForHelp = async (
  to: RouteLocationNormalized
) => {
  const { name } = to;
  const uiStateStore = useUIStateStore();
  const helpStore = useHelpStore();

  if (!routeMap) {
    const res = await fetch("/help/routeMap.json");
    routeMap = await res.json();
  }

  const helpName = routeMap?.find((pair) => pair.routeName === name)?.helpName;

  if (helpName) {
    if (!uiStateStore.getIntroStateByKey(`guide.${helpName}`)) {
      if (timer) {
        clearTimeout(timer);
      }

      timer = window.setTimeout(() => {
        helpStore.showHelp(helpName, true);
        uiStateStore.saveIntroStateByKey({
          key: "environment.visit",
          newState: true,
        });
      }, 1000);
    }
  }
};
