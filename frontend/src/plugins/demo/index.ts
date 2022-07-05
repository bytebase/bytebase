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
