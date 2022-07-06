import { indexOf, isUndefined } from "lodash-es";
import { getStringFromI18NText } from "./i18n";
import { DialogPosition, HintData } from "./types";
import {
  checkUrlPathnameMatched,
  getElementBounding,
  getElementMaxZIndex,
  waitForTargetElement,
} from "./utils";

const closedHintIndexSet = new Set<number>();

export const showHints = async (hintDataList: HintData[]) => {
  removeHint();
  for (const hintData of hintDataList) {
    const index = indexOf(hintDataList, hintData);
    if (
      closedHintIndexSet.has(index) ||
      (hintData.url && !checkUrlPathnameMatched(hintData.url))
    ) {
      continue;
    }

    const targetElement = await waitForTargetElement([[hintData.selector]]);
    if (targetElement) {
      removeHint(index);
      renderHighlight(targetElement, hintData, index);
      updateHintPosition(targetElement, hintData, index);
    }
  }
};

const renderHighlight = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  const highlightWrapper = document.createElement("div");
  highlightWrapper.className = `bb-hint-highlight-wrapper bb-hint-${hintIndex}`;
  document.body.appendChild(highlightWrapper);

  const maxZIndex = getElementMaxZIndex(targetElement);
  highlightWrapper.style.zIndex = `${maxZIndex}`;

  if (hintData.additionStyle) {
    for (const key in hintData.additionStyle) {
      highlightWrapper.style[key] = hintData.additionStyle[key];
    }
  }

  if (!checkShouldShowDialog(hintData)) {
    highlightWrapper.style.pointerEvents = "none";
  }

  const bounding = getElementBounding(targetElement);
  highlightWrapper.style.top = `${bounding.top}px`;
  highlightWrapper.style.left = `${bounding.left}px`;
  highlightWrapper.style.width = `${bounding.width}px`;
  highlightWrapper.style.height = `${bounding.height}px`;

  highlightWrapper.addEventListener(
    "click",
    (e) => {
      e.stopPropagation();
      highlightWrapper.style.pointerEvents = "none";
      renderHintDialog(targetElement, hintData, hintIndex);
    },
    {
      once: true,
    }
  );
};

const renderHintDialog = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  const { title, description } = hintData;
  const hintDialogDiv = document.createElement("div");
  hintDialogDiv.className = `bb-hint-dialog bb-hint-dialog-${hintIndex}`;

  const maxZIndex = getElementMaxZIndex(targetElement);
  hintDialogDiv.style.zIndex = `${maxZIndex}`;

  if (hintData.additionStyle) {
    for (const key in hintData.additionStyle) {
      hintDialogDiv.style[key] = hintData.additionStyle[key];
    }
  }

  const closeButton = document.createElement("button");
  closeButton.className = "bb-hint-close-button";
  closeButton.innerHTML = "&times;";
  closeButton.addEventListener("click", () => {
    removeHint(hintIndex);
    closedHintIndexSet.add(hintIndex);
  });
  hintDialogDiv.appendChild(closeButton);

  if (getStringFromI18NText(title)) {
    const titleElement = document.createElement("p");
    titleElement.className = "bb-hint-title-text";
    titleElement.innerText = getStringFromI18NText(title);
    hintDialogDiv.appendChild(titleElement);
  }
  if (getStringFromI18NText(description)) {
    const descriptionElement = document.createElement("p");
    descriptionElement.className = "bb-hint-description-text";
    descriptionElement.innerText = getStringFromI18NText(description);
    hintDialogDiv.appendChild(descriptionElement);
  }

  document.body.appendChild(hintDialogDiv);
  adjustHintDialogPosition(targetElement, hintDialogDiv, hintData.position);
};

const adjustHintDialogPosition = (
  targetElement: HTMLElement,
  hintDialogDiv: HTMLElement,
  position: DialogPosition = "bottom"
) => {
  const bounding = getElementBounding(targetElement);
  const guideDialogBounding = getElementBounding(hintDialogDiv);

  if (position === "bottom") {
    hintDialogDiv.style.top = `${bounding.top + bounding.height}px`;
    hintDialogDiv.style.left = `${bounding.left - 4}px`;
  } else if (position === "top") {
    hintDialogDiv.style.top = `${
      bounding.top - guideDialogBounding.height - 8
    }px`;
    hintDialogDiv.style.left = `${bounding.left - 4}px`;
  } else if (position === "left") {
    hintDialogDiv.style.top = `${bounding.top - 4}px`;
    hintDialogDiv.style.left = `${
      bounding.left - guideDialogBounding.width - 8
    }px`;
  } else if (position === "right") {
    hintDialogDiv.style.top = `${bounding.top - 4}px`;
    hintDialogDiv.style.left = `${bounding.left + bounding.width}px`;
  }
};

const updateHintPosition = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  const highlightWrapper = document.body.querySelector(
    `.bb-hint-highlight-wrapper.bb-hint-${hintIndex}`
  ) as HTMLElement;

  if (!targetElement || !highlightWrapper) {
    removeHint(hintIndex);
    return;
  }

  const bounding = getElementBounding(targetElement);

  highlightWrapper.style.top = `${bounding.top}px`;
  highlightWrapper.style.left = `${bounding.left}px`;
  highlightWrapper.style.width = `${bounding.width}px`;
  highlightWrapper.style.height = `${bounding.height}px`;

  requestAnimationFrame(() =>
    updateHintPosition(targetElement, hintData, hintIndex)
  );
};

export const removeHint = (hintIndex?: number) => {
  if (isUndefined(hintIndex)) {
    const hintWrappers = document.querySelectorAll(
      ".bb-hint-highlight-wrapper"
    );
    if (hintWrappers) {
      hintWrappers.forEach((hintWrapper) => {
        hintWrapper.remove();
      });
    }
    const hintDialogs = document.querySelectorAll(".bb-hint-dialog");
    if (hintDialogs) {
      hintDialogs.forEach((hintDialog) => {
        hintDialog.remove();
      });
    }
  } else {
    const hintWrapper = document.querySelector(
      `.bb-hint-highlight-wrapper.bb-hint-${hintIndex}`
    );
    if (hintWrapper) {
      hintWrapper.remove();
    }
    const hintDialog = document.querySelector(
      `.bb-hint-dialog.bb-hint-dialog-${hintIndex}`
    );
    if (hintDialog) {
      hintDialog.remove();
    }
  }
};

const checkShouldShowDialog = (hintData: HintData): boolean => {
  if (!isUndefined(hintData.title)) {
    return true;
  }

  return false;
};
