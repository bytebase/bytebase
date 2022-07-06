import { isNull } from "lodash-es";
import { initGuideListeners, initHintListeners } from "./listener";
import * as storage from "./storage";

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
    }
    if (paramHint || storageHint) {
      initHintListeners();
    }
  },
  {
    once: true,
  }
);

export default null;
