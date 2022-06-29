import { merge } from "lodash-es";
import { getElementBounding, waitForTargetElement } from "./utils";
import * as storage from "./storage";
import { GuidePosition, StepData } from "./types";

export const showGuideDialog = async (guideStep: StepData) => {
  removeGuideDialog();
  const targetElement = await waitForTargetElement(guideStep.selectors);
  if (targetElement) {
    // After got the targetElement, remove the guide dialog again
    // to ensure that only one guide dialog is shown at a time.
    removeGuideDialog();
    if (guideStep.cover) {
      renderCover();
      targetElement.classList.add("bb-guide-target-element");
    }
    renderHighlight(targetElement);
    renderGuideDialog(targetElement, guideStep);
    requestAnimationFrame(() => updateGuidePosition(targetElement, guideStep));
  }
};

const renderCover = () => {
  const coverElement = document.createElement("div");
  coverElement.className = "bb-guide-cover-wrapper";
  document.body.appendChild(coverElement);
};

const renderHighlight = (targetElement: HTMLElement) => {
  const highlightWrapper = document.createElement("div");
  highlightWrapper.className = "bb-guide-highlight-wrapper";
  document.body.appendChild(highlightWrapper);
  const bounding = getElementBounding(targetElement);
  highlightWrapper.style.top = `${bounding.top}px`;
  highlightWrapper.style.left = `${bounding.left}px`;
  highlightWrapper.style.width = `${bounding.width}px`;
  highlightWrapper.style.height = `${bounding.height}px`;
};

const renderGuideDialog = (targetElement: HTMLElement, guideStep: StepData) => {
  const { description, title, type } = guideStep;
  const guideDialogDiv = document.createElement("div");
  guideDialogDiv.className = "bb-guide-dialog";
  adjustGuideDialogPosition(targetElement, guideDialogDiv, guideStep.position);

  const closeButton = document.createElement("button");
  closeButton.className = "bb-guide-close-button";
  closeButton.innerHTML = "&times;";
  closeButton.addEventListener("click", () => removeGuideDialog());
  guideDialogDiv.appendChild(closeButton);

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

  const nextButton = document.createElement("button");
  nextButton.className = "button";
  nextButton.innerText = "Next";
  buttonsContainer.appendChild(nextButton);
  guideDialogDiv.appendChild(buttonsContainer);

  if (type === "click") {
    targetElement.addEventListener("click", () => {
      const { guide } = storage.get(["guide"]);
      storage.set({
        guide: merge(guide, {
          stepIndex: (guide?.stepIndex ?? 0) + 1,
        }),
      });
      storage.emitStorageChangedEvent();
    });
    nextButton.onclick = () => {
      targetElement.click();
    };
  } else if (type === "change") {
    nextButton.onclick = () => {
      const value =
        targetElement.textContent ||
        targetElement.nodeValue ||
        (targetElement as HTMLInputElement).value ||
        "";
      if (guideStep.value && RegExp(guideStep.value).test(value)) {
        const { guide } = storage.get(["guide"]);
        storage.set({
          guide: merge(guide, {
            stepIndex: (guide?.stepIndex ?? 0) + 1,
          }),
        });
        storage.emitStorageChangedEvent();
      } else {
        // ...show invalid value message
      }
    };
  }

  document.body.appendChild(guideDialogDiv);
};

const adjustGuideDialogPosition = (
  targetElement: HTMLElement,
  guideDialogDiv: HTMLElement,
  position: GuidePosition = "bottom"
) => {
  const bounding = getElementBounding(targetElement);
  const guideDialogBounding = getElementBounding(guideDialogDiv);
  if (position === "bottom") {
    guideDialogDiv.style.top = `${bounding.top + bounding.height}px`;
    guideDialogDiv.style.left = `${bounding.left - 4}px`;
  } else if (position === "top") {
    guideDialogDiv.style.top = `${
      bounding.top - guideDialogBounding.height - 8
    }px`;
    guideDialogDiv.style.left = `${bounding.left - 4}px`;
  } else if (position === "left") {
    guideDialogDiv.style.top = `${bounding.top - 4}px`;
    guideDialogDiv.style.left = `${
      bounding.left - guideDialogBounding.width - 8
    }px`;
  } else if (position === "right") {
    guideDialogDiv.style.top = `${bounding.top - 4}px`;
    guideDialogDiv.style.left = `${bounding.left + bounding.width}px`;
  }
};

const updateGuidePosition = (
  targetElement: HTMLElement,
  guideStep: StepData
) => {
  const highlightWrapper = document.body.querySelector(
    ".bb-guide-highlight-wrapper"
  ) as HTMLElement;
  const guideDialogDiv = document.body.querySelector(
    ".bb-guide-dialog"
  ) as HTMLElement;

  if (!targetElement || !highlightWrapper || !guideDialogDiv) {
    return;
  }

  const bounding = getElementBounding(targetElement);

  highlightWrapper.style.top = `${bounding.top}px`;
  highlightWrapper.style.left = `${bounding.left}px`;
  highlightWrapper.style.width = `${bounding.width}px`;
  highlightWrapper.style.height = `${bounding.height}px`;

  adjustGuideDialogPosition(targetElement, guideDialogDiv, guideStep.position);

  requestAnimationFrame(() => updateGuidePosition(targetElement, guideStep));
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
  document.body
    .querySelectorAll(".bb-guide-cover-wrapper")
    ?.forEach((element) => {
      element.remove();
    });
  document.body
    .querySelectorAll(".bb-guide-target-element")
    ?.forEach((element) => {
      element.classList.remove("bb-guide-target-element");
    });
};
