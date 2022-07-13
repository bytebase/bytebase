import { isEqual } from "lodash-es";
import { fetchGuideDataWithName, fetchHintDataWithName } from "./data";
import { removeGuideDialog, showGuideDialog } from "./guide";
import { removeHint, showHints } from "./hint";
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

const refreshLocalStorageHintData = async () => {
  const params = new URLSearchParams(window.location.search);
  const hintName = params.get("hint");

  if (hintName) {
    const hintData = await fetchHintDataWithName(hintName);

    if (hintData) {
      storage.set({
        hint: {
          name: hintName,
        },
      });
    }
  }
};

const tryToShowGuideDialog = async () => {
  const { guide } = storage.get(["guide"]);

  if (guide) {
    const guideData = await fetchGuideDataWithName(guide.name);
    showGuideDialog(guideData, guide.stepIndex);
  } else {
    removeGuideDialog();
  }
};

const tryToShowHints = async () => {
  const { hint } = storage.get(["hint"]);

  if (hint) {
    const hintData = await fetchHintDataWithName(hint.name);
    showHints(hintData);
  } else {
    removeHint();
  }
};

// initLocationListenerForGuide using mutation observer to detect DOM changes for mocking location changed.
const initLocationListenerForGuide = () => {
  let prevLocationString = "";

  const observer = new MutationObserver(async () => {
    await refreshLocalStorageGuideData();
    const locationString = window.location.toString();

    if (!isEqual(prevLocationString, locationString)) {
      prevLocationString = locationString;
      tryToShowGuideDialog();
    }
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });
};

// initMutationListenerForHint using mutation observer to detect DOM changes.
const initMutationListenerForHint = () => {
  const observer = new MutationObserver(async (mutations: MutationRecord[]) => {
    for (const mutation of mutations) {
      const changedNodes = [...mutation.addedNodes, ...mutation.removedNodes];
      for (const changedNode of changedNodes) {
        if (
          changedNode instanceof HTMLElement &&
          !changedNode.matches(".bb-hint-wrapper") &&
          !changedNode.matches(".bb-hint-cover-wrapper") &&
          !changedNode.matches(".bb-hint-dialog")
        ) {
          tryToShowHints();
          return;
        }
      }
    }
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });
};

export const initGuideListeners = () => {
  initLocationListenerForGuide();
  window.addEventListener("storage", async () => {
    tryToShowGuideDialog();
  });
};

export const initHintListeners = async () => {
  await refreshLocalStorageHintData();
  initMutationListenerForHint();
  window.addEventListener("storage", async () => {
    tryToShowHints();
  });
};
