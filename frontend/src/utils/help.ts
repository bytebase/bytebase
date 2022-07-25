import { RouteLocationNormalized } from "vue-router";
import { useUIStateStore, useHelpStore } from "@/store";

export const helpOnRouteChange = async (to: RouteLocationNormalized) => {
  const { name } = to;
  const uiStateStore = useUIStateStore();
  const helpStore = useHelpStore();
  const res = await fetch("/help/routeMap.json");
  const routeMap = await res.json();

  if (routeMap[name as keyof typeof routeMap]) {
    if (
      !uiStateStore.getIntroStateByKey(
        `guide.${routeMap[name as keyof typeof routeMap]}`
      )
    ) {
      if (helpStore.timer) {
        clearTimeout(helpStore.timer);
      }
      helpStore.setTimer(
        window.setTimeout(() => {
          helpStore.showHelp(routeMap[name as keyof typeof routeMap], true);
          uiStateStore.saveIntroStateByKey({
            key: "environment.visit",
            newState: true,
          });
        }, 1000)
      );
    }
  }
};
