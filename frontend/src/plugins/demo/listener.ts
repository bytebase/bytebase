import { isEqual } from "lodash-es";
import { fetchDemoDataWithName, fetchGuideDataWithName } from "./data";
import { removeGuideDialog, showGuideDialog } from "./guide";
import { removeHint, showHints } from "./hint";
import * as storage from "./storage";

const tryToShowGuides = async () => {
  const { guide } = storage.get(["guide"]);

  if (guide) {
    const guideData = await fetchGuideDataWithName(guide.name);
    showGuideDialog(guideData, guide.stepIndex);
  } else {
    removeGuideDialog();
  }
};

const tryToShowHints = async () => {
  const { demo } = storage.get(["demo"]);

  if (demo) {
    const demoData = await fetchDemoDataWithName(demo.name);
    showHints(demoData.hint);
  } else {
    removeHint();
  }
};

// initLocationListenerForGuide using mutation observer to detect DOM changes for mocking location changed.
const initLocationListenerForGuide = () => {
  let prevLocationString = "";

  const observer = new MutationObserver(async () => {
    const locationString = window.location.toString();

    if (!isEqual(prevLocationString, locationString)) {
      prevLocationString = locationString;
      tryToShowGuides();
    }
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });
};

// initMutationListenerForHint using mutation observer to detect DOM changes.
const initMutationListenerForHint = () => {
  const observer = new MutationObserver((mutations: MutationRecord[]) => {
    setTimeout(async () => {
      for (const mutation of mutations) {
        const changedNodes = [...mutation.addedNodes, ...mutation.removedNodes];
        for (const changedNode of changedNodes) {
          if (
            changedNode instanceof HTMLElement &&
            !changedNode.matches(".bb-demo-element")
          ) {
            await tryToShowHints();
            return;
          }
        }
      }
    });
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });
};

const initGuideListeners = () => {
  initLocationListenerForGuide();
  window.addEventListener("storage", async () => {
    tryToShowGuides();
  });
};

const initHintListeners = () => {
  initMutationListenerForHint();
  window.addEventListener("storage", async () => {
    tryToShowHints();
  });
};

const initDemoListeners = async () => {
  const { demo } = storage.get(["demo"]);
  if (demo && demo.name) {
    const demoData = await fetchDemoDataWithName(demo.name);
    if (demoData) {
      try {
        const guideData = await fetchGuideDataWithName(demo.name);
        if (guideData) {
          await tryToShowGuides();
          initGuideListeners();
        }
      } catch (error) {
        // do nth
      }

      try {
        const demoData = await fetchDemoDataWithName(demo.name);
        if (demoData.hint) {
          await tryToShowHints();
          initHintListeners();
        }
      } catch (error) {
        // do nth
      }
    }
  }
};

export const initLocationListenerForDemo = async () => {
  const { demo } = storage.get(["demo"]);

  if (demo && demo.name) {
    const demoData = await fetchDemoDataWithName(demo.name);
    if (demoData) {
      let prevLocationString = "";

      const observer = new MutationObserver(async () => {
        const locationString = window.location.toString();

        if (!isEqual(prevLocationString, locationString)) {
          prevLocationString = locationString;
          // Do not show demo element in /auth pages.
          if (window.location.pathname.startsWith("/auth")) {
            return;
          }

          initDemoListeners();
          observer.disconnect();
        }
      });

      observer.observe(document.body, {
        childList: true,
        subtree: true,
      });
    }
  }
};
