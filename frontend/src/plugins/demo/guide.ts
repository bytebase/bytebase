import { merge } from "lodash-es";
import { getStringFromI18NText, nextI18NText } from "./i18n";
import * as storage from "./storage";
import { Position, GuideData, StepData } from "./types";
import {
  checkUrlPathnameMatched,
  getElementBounding,
  getElementMaxZIndex,
  isNullOrUndefined,
  waitForTargetElement,
} from "./utils";

// validateStepData will check if the step data is valid
export const validateStepData = (stepData: StepData) => {
  // type is required and must be one of ["click", "change"]
  if (stepData.type !== "click" && stepData.type !== "change") {
    return false;
  }
  // title is required
  if (isNullOrUndefined(stepData.title)) {
    return false;
  }
  // description is required
  if (isNullOrUndefined(stepData.description)) {
    return false;
  }

  return true;
};

export const showGuideDialog = async (
  guideData: GuideData,
  stepIndex: number
) => {
  removeGuideDialog();
  const guideStep = guideData.steps[stepIndex];
  if (!guideStep) {
    return;
  }

  if (guideStep.url && !checkUrlPathnameMatched(guideStep.url)) {
    return;
  }

  // NOTE: stay empty value for now.
  const targetElement = await waitForTargetElement("");
  if (targetElement) {
    // After got the targetElement, remove the guide dialog again
    // to ensure that only one guide dialog is shown at a time.
    removeGuideDialog();
    renderHighlight(targetElement, guideStep);
    renderGuideDialog(targetElement, guideData, stepIndex);
    requestAnimationFrame(() => updateDialogPosition(targetElement, guideStep));
  }
};

const renderHighlight = (targetElement: HTMLElement, guideStep: StepData) => {
  const highlightWrapper = document.createElement("div");
  highlightWrapper.className = "bb-demo-element bb-guide-highlight-wrapper";
  document.body.appendChild(highlightWrapper);
  const bounding = getElementBounding(targetElement);
  highlightWrapper.style.top = `${bounding.top}px`;
  highlightWrapper.style.left = `${bounding.left}px`;
  highlightWrapper.style.width = `${bounding.width}px`;
  highlightWrapper.style.height = `${bounding.height}px`;

  if (guideStep.cover) {
    const maxZIndex = getElementMaxZIndex(targetElement);
    highlightWrapper.classList.add("covered");
    const coverElement = document.createElement("div");
    coverElement.className = "bb-demo-element bb-guide-cover-wrapper";
    coverElement.style.zIndex = `${maxZIndex - 1}`;
    document.body.appendChild(coverElement);
    targetElement.classList.add("bb-guide-target-element");
  }
};

const renderGuideDialog = (
  targetElement: HTMLElement,
  guideData: GuideData,
  stepIndex: number
) => {
  const guideStep = guideData.steps[stepIndex];
  const { description, title, type } = guideStep;
  const guideDialogDiv = document.createElement("div");
  guideDialogDiv.className = "bb-demo-element bb-guide-dialog";
  adjustGuideDialogPosition(targetElement, guideDialogDiv, guideStep.position);

  const closeButton = document.createElement("button");
  closeButton.className = "bb-demo-element bb-guide-close-button";
  closeButton.innerHTML = "&times;";
  closeButton.addEventListener("click", () => removeGuideDialog());
  guideDialogDiv.appendChild(closeButton);

  const titleElement = document.createElement("p");
  titleElement.className = "bb-demo-element bb-guide-title-text";
  titleElement.innerText = getStringFromI18NText(title);
  guideDialogDiv.appendChild(titleElement);
  const descriptionElement = document.createElement("p");
  descriptionElement.className = "bb-demo-element bb-guide-description-text";
  descriptionElement.innerText = getStringFromI18NText(description);
  guideDialogDiv.appendChild(descriptionElement);

  if (!guideStep.hideNextButton) {
    const buttonsContainer = document.createElement("div");
    buttonsContainer.className = "bb-demo-element bb-guide-btns-container";
    const nextButton = document.createElement("button");
    nextButton.className = "bb-demo-element button";
    if (stepIndex === guideData.steps.length - 1) {
      nextButton.innerText = "Done";
    } else {
      nextButton.innerText = getStringFromI18NText(nextI18NText);
    }
    buttonsContainer.appendChild(nextButton);
    guideDialogDiv.appendChild(buttonsContainer);

    if (type === "click") {
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
              stepIndex: stepIndex + 1,
            }),
          });
          storage.emitStorageChangedEvent();
        } else {
          // show invalid value message by alert
          alert(`invalid content value, should be like \`${guideStep.value}\``);
        }
      };
    }
  }

  if (type === "click") {
    targetElement.addEventListener("click", () => {
      const { guide } = storage.get(["guide"]);
      storage.set({
        guide: merge(guide, {
          stepIndex: stepIndex + 1,
        }),
      });
      if (stepIndex + 1 >= guideData.steps.length) {
        storage.remove(["guide"]);
      }
      storage.emitStorageChangedEvent();
    });
  }

  document.body.appendChild(guideDialogDiv);
};

const adjustGuideDialogPosition = (
  targetElement: HTMLElement,
  guideDialogDiv: HTMLElement,
  position: Position = "bottom"
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

const updateDialogPosition = (
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

  requestAnimationFrame(() => updateDialogPosition(targetElement, guideStep));
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
