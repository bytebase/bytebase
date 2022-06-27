import { getElementBounding } from "./utils";
import * as storage from "./storage";
import { merge } from "lodash-es";

const getTargetElementBySelectors = (selectors: string[][]) => {
  let targetElement = document.body;
  for (const selector of selectors) {
    try {
      targetElement = document.body.querySelector(
        selector.join(" ")
      ) as HTMLElement;
    } catch (error) {
      // do nth
    }

    if (targetElement) {
      break;
    }
  }
  return targetElement;
};

export const showGuideDialog = async (guideStep: any) => {
  removeGuideDialog();
  const targetElement = await waitForTargetElement(guideStep.selectors);
  if (targetElement) {
    renderHighlightWrapper(targetElement);
    renderGuideDialog(targetElement, guideStep.title, guideStep.description);
  }
};

const renderHighlightWrapper = (targetElement: HTMLElement) => {
  const highlightWrapper = document.createElement("div");
  highlightWrapper.className = "bb-guide-highlight-wrapper";
  document.body.appendChild(highlightWrapper);
  const bounding = getElementBounding(targetElement);
  highlightWrapper.style.top = `${bounding.top}px`;
  highlightWrapper.style.left = `${bounding.left}px`;
  highlightWrapper.style.width = `${bounding.width}px`;
  highlightWrapper.style.height = `${bounding.height}px`;
};

const renderGuideDialog = (
  targetElement: HTMLElement,
  title: string,
  description: string
) => {
  const guideDialogDiv = document.createElement("div");
  guideDialogDiv.className = "bb-guide-dialog";
  const bounding = getElementBounding(targetElement);
  guideDialogDiv.style.top = `${bounding.top + bounding.height}px`;
  guideDialogDiv.style.left = `${bounding.left}px`;
  const titleElement = document.createElement("p");
  titleElement.className = "bb-guide-title-text";
  titleElement.innerText = title;
  guideDialogDiv.appendChild(titleElement);
  const descriptionElement = document.createElement("p");
  descriptionElement.className = "bb-guide-description-text";
  descriptionElement.innerText = description;
  guideDialogDiv.appendChild(descriptionElement);

  const buttonsContainer = document.createElement("div");
  buttonsContainer.className = "bb-guide-btns-container";
  const prevButton = document.createElement("button");
  prevButton.className = "button";
  prevButton.innerText = "Back";
  const nextButton = document.createElement("button");
  nextButton.className = "button";
  nextButton.innerText = "Next";
  buttonsContainer.appendChild(prevButton);
  buttonsContainer.appendChild(nextButton);
  guideDialogDiv.appendChild(buttonsContainer);

  prevButton.onclick = () => {
    // ...how to do prev?
  };

  nextButton.onclick = () => {
    targetElement.click();
  };

  // TOOD(steven): support more action type: input value change...
  targetElement.addEventListener("click", () => {
    const { guide } = storage.get(["guide"]);
    storage.set({
      guide: merge(guide, {
        stepIndex: (guide?.stepIndex ?? 0) + 1,
      }),
    });
    storage.emitStorageChangedEvent();
  });

  document.body.appendChild(guideDialogDiv);
};

const waitForTargetElement = (selectors: string[][]): Promise<HTMLElement> => {
  return new Promise((resolve) => {
    let targetElement = getTargetElementBySelectors(selectors);
    if (targetElement) {
      return resolve(targetElement);
    }

    const observer = new MutationObserver(() => {
      targetElement = getTargetElementBySelectors(selectors);
      if (targetElement) {
        observer.disconnect();
        return resolve(targetElement);
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true,
    });
  });
};

export const removeGuideDialog = () => {
  document.body.querySelectorAll(".bb-guide-dialog")?.forEach((element) => {
    element.remove();
  });
  document.body
    .querySelectorAll(".bb-guide-highlight-wrapper")
    ?.forEach((element) => {
      element.remove();
    });
};
