import { isEqual } from "lodash-es";
import { fetchGuideDataWithName } from "./data";
import { removeGuideDialog, showGuideDialog } from "./dialog";
import * as storage from "./storage";

const refreshLocalStorageGuideData = async () => {
  const params = new URLSearchParams(window.location.search);
  const guideName = params.get("guide");
  const { guide } = storage.get(["guide"]);

  if (guideName && guideName !== guide?.name) {
    const guideData = await fetchGuideDataWithName(guideName);

    if (guideData) {
      storage.set({
        guide: {
          name: guideName,
          stepIndex: 0,
        },
      });
    }
  }
};

const tryToShowGuideDialog = async () => {
  const { guide } = storage.get(["guide"]);

  if (guide) {
    const guideData = await fetchGuideDataWithName(guide.name);
    const stepData = guideData.steps[guide.stepIndex];
    if (stepData) {
      showGuideDialog(stepData);
    }
  } else {
    removeGuideDialog();
  }
};

// initLocationListener using mutation observer to detect DOM changes for mocking location changed.
const initLocationListener = () => {
  let prevLocationString = "";

  const observer = new MutationObserver(async () => {
    await refreshLocalStorageGuideData();
    const locationString = window.location.toString();

    if (!isEqual(prevLocationString, locationString)) {
      prevLocationString = locationString;
      tryToShowGuideDialog();
    }
  });

  observer.observe(document.querySelector("body") as HTMLBodyElement, {
    childList: true,
    subtree: true,
  });
};

// initStorageListener detecting storage changes
const initStorageListener = () => {
  window.addEventListener("storage", async () => {
    tryToShowGuideDialog();
  });
};

const initGuideListeners = () => {
  initLocationListener();
  initStorageListener();
};

export default initGuideListeners;
