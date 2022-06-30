import initGuideListeners from "./listener";
import * as storage from "./storage";

// initial guide listeners when window loaded
window.addEventListener(
  "load",
  () => {
    const params = new URLSearchParams(window.location.search);
    const paramGuide = params.get("guide");
    const { guide: storageGuide } = storage.get(["guide"]);

    if (paramGuide || storageGuide) {
      initGuideListeners();
    }
  },
  {
    once: true,
  }
);

export default null;
