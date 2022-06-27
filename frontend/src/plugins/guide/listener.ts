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

    // remove useless type guideData
    guideData.steps = guideData.steps.filter(
      (s: any) => s.type !== "setViewport" && s.type !== "navigate"
    );

    showGuideDialog(guideData.steps[guide.stepIndex]);
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
    console.log("storage changed");
    tryToShowGuideDialog();
  });
};

const initGuideListeners = () => {
  console.log("init listeners");
  initLocationListener();
  initStorageListener();
};

export default initGuideListeners;
