import { isNull } from "lodash-es";
import { initGuideListeners, initHintListeners } from "./listener";
import * as storage from "./storage";

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
  () => {
    const { guide: storageGuide, hint: storageHint } = storage.get([
      "guide",
      "hint",
    ]);
    const params = new URLSearchParams(window.location.search);

    // If there is a cleardemo flag in the url, then clear the demo storage and return
    const clearDemo = params.get("cleardemo");
    if (!isNull(clearDemo)) {
      storage.remove(["guide", "hint"]);
      return;
    }

    const paramGuide = params.get("guide");
    const paramHint = params.get("hint");

    if (paramGuide || storageGuide) {
      initGuideListeners();
      hideConsoleGuides();
    }
    if (paramHint || storageHint) {
      initHintListeners();
      hideConsoleGuides();
    }
  },
  {
    once: true,
  }
);

export default null;
