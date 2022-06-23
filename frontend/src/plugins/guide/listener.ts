import { isEqual } from "lodash-es";
import { fetchGuideDataWithName } from "./data";
import * as storage from "./storage";

// initLocationListener using mutation observer to detect DOM changes for mocking location changed.
const initLocationListener = () => {
  let prevLocationString = "";

  const observer = new MutationObserver(async () => {
    const locationString = window.location.toString();

    if (!isEqual(prevLocationString, locationString)) {
      prevLocationString = locationString;
      let { guide } = storage.get(["guide"]);

      const params = new URLSearchParams(window.location.search);
      const guideName = params.get("guide");

      if (guideName && guideName !== guide?.name) {
        const guideData = await fetchGuideDataWithName(guideName);

        if (guideData) {
          const tempGuide = {
            name: guideName,
            startRoute: "",
            stepIndex: 0,
          };

          guide = tempGuide;
          storage.set({
            guide: tempGuide,
          });
        }
      }

      if (guide) {
        const guideData = await fetchGuideDataWithName(guide.name);
        console.log("guideData", guideData);
        // ...do something with the guide and guideData
      }
    }
  });

  observer.observe(document.querySelector("body") as HTMLBodyElement, {
    childList: true,
    subtree: true,
  });
};

// initStorageListener detecting storage changes
const initStorageListener = () => {
  window.addEventListener("storage", () => {
    console.log("storage changed");
  });
};

const initGuideListeners = () => {
  console.log("init listeners");
  initLocationListener();
  initStorageListener();
};

export default initGuideListeners;
