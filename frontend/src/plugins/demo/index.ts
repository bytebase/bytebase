import axios from "axios";
import { isNull } from "lodash-es";
import { initGuideListeners, initHintListeners } from "./listener";
import * as storage from "./storage";

const invalidDemoNameList = ["dev", "prod"];

// TODO(steven): after we using help to replace the guide, we should remove this.
// In live demo mode, we should not show the guide dialog.
const hideConsoleGuides = () => {
  const tempData = {
    "database.visit": true,
    "instance.visit": true,
    "project.visit": true,
    "environment.visit": true,
    "guide.database": true,
    "guide.instance": true,
    "guide.project": true,
    "guide.environment": true,
  };
  window.localStorage.setItem("ui.intro", JSON.stringify(tempData));
};

// initial guide listeners when window loaded
window.addEventListener(
  "load",
  async () => {
    const serverInfo = (
      await axios.get<{
        demo: boolean;
        demoName: string;
      }>(`/api/actuator/info`)
    ).data;

    // In online demo, we only show the hints.
    if (
      serverInfo.demo &&
      serverInfo.demoName &&
      !invalidDemoNameList.includes(serverInfo.demoName)
    ) {
      const demoName = serverInfo.demoName;
      if (demoName) {
        storage.set({
          hint: {
            name: demoName,
          },
        });
        hideConsoleGuides();
        initHintListeners();
      }
    } else {
      // This is using for dev mainly.
      const { guide: storageGuide, hint: storageHint } = storage.get([
        "guide",
        "hint",
      ]);
      const params = new URLSearchParams(window.location.search);
      const paramGuide = params.get("guide");
      const paramHint = params.get("hint");

      if (paramGuide || storageGuide) {
        hideConsoleGuides();
        initGuideListeners();
      }
      if (paramHint || storageHint) {
        hideConsoleGuides();
        initHintListeners();
      }
    }

    // If there is a `cleardemo` flag in the url, then clear the demo storage.
    const params = new URLSearchParams(window.location.search);
    const clearDemo = params.get("cleardemo");
    if (!isNull(clearDemo)) {
      storage.remove(["guide", "hint"]);
    }
  },
  {
    once: true,
  }
);

export default null;
