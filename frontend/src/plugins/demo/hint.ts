import { indexOf, isUndefined } from "lodash-es";
import { getStringFromI18NText } from "./i18n";
import { DialogPosition, HintData } from "./types";
import {
  checkUrlMatched,
  getElementBounding,
  getElementMaxZIndex,
  waitForTargetElement,
} from "./utils";

const shownHintIndexSet = new Set<number>();
const closedHintIndexSet = new Set<number>();

export const showHints = async (hintDataList: HintData[]) => {
  removeHint();
  shownHintIndexSet.clear();

  const findTargetPromiseList = hintDataList.map(async (hintData) => {
    const index = indexOf(hintDataList, hintData);
    if (closedHintIndexSet.has(index) || !checkUrlMatched(hintData.url)) {
      return;
    }

    const targetElement = await waitForTargetElement([[hintData.selector]]);
    if (targetElement) {
      if (shownHintIndexSet.has(index)) {
        return;
      }
      renderHint(targetElement, hintData, index);
      updateHintPosition(targetElement, hintData, index);
      shownHintIndexSet.add(index);
    } else {
      removeHint(index);
      shownHintIndexSet.delete(index);
    }
  });

  Promise.all(findTargetPromiseList);
};

const renderHint = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  const hintWrapper = document.createElement("div");
  hintWrapper.className = `bb-hint-wrapper bb-hint-${hintIndex}`;
  if (hintData.highlight) {
    hintWrapper.classList.add("highlight");
  }
  if (hintData.type === "shield") {
    hintWrapper.classList.add("shield");
  }
  document.body.appendChild(hintWrapper);

  const maxZIndex = getElementMaxZIndex(targetElement);
  hintWrapper.style.zIndex = `${maxZIndex}`;

  if (hintData.additionStyle) {
    for (const key in hintData.additionStyle) {
      hintWrapper.style[key] = hintData.additionStyle[key];
    }
  }

  const bounding = getElementBounding(targetElement);
  hintWrapper.style.top = `${bounding.top}px`;
  hintWrapper.style.left = `${bounding.left}px`;
  hintWrapper.style.width = `${bounding.width}px`;
  hintWrapper.style.height = `${bounding.height}px`;

  if (isUndefined(hintData.dialog)) {
    hintWrapper.style.pointerEvents = "none";
  } else {
    if (hintData.dialog.alwaysShow) {
      renderHintDialog(targetElement, hintData, hintIndex);
      hintWrapper.style.pointerEvents = "none";
    } else {
      hintWrapper.addEventListener(
        "click",
        (e) => {
          e.stopPropagation();
          hintWrapper.style.pointerEvents = "none";
          if (!isUndefined(hintData.dialog)) {
            if (hintData.dialog.showOnce) {
              closedHintIndexSet.add(hintIndex);
            }
            renderHintDialog(targetElement, hintData, hintIndex);
          }
        },
        {
          once: true,
        }
      );
    }
  }

  if (hintData.cover) {
    const maxZIndex = getElementMaxZIndex(targetElement);
    hintWrapper.classList.add("covered");
    const coverElement = document.createElement("div");
    coverElement.className = `bb-hint-cover-wrapper bb-hint-${hintIndex}`;
    coverElement.style.zIndex = `${Math.max(maxZIndex - 1, 0)}`;
    document.body.appendChild(coverElement);
    targetElement.classList.add("bb-hint-target-element");
  }
};

const renderHintDialog = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  if (isUndefined(hintData.dialog)) {
    return;
  }

  const {
    dialog: { title, description, position, alwaysShow },
  } = hintData;
  const hintDialogDiv = document.createElement("div");
  hintDialogDiv.className = `bb-hint-dialog bb-hint-dialog-${hintIndex}`;

  const maxZIndex = getElementMaxZIndex(targetElement);
  hintDialogDiv.style.zIndex = `${maxZIndex}`;

  if (hintData.additionStyle) {
    for (const key in hintData.additionStyle) {
      hintDialogDiv.style[key] = hintData.additionStyle[key];
    }
  }

  if (!alwaysShow) {
    const closeButton = document.createElement("button");
    closeButton.className = "bb-hint-close-button";
    closeButton.innerHTML = "&times;";
    closeButton.addEventListener("click", () => {
      removeHint(hintIndex);
      closedHintIndexSet.add(hintIndex);
    });
    hintDialogDiv.appendChild(closeButton);
  }

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
  adjustHintDialogPosition(targetElement, hintDialogDiv, position);
};

const adjustHintDialogPosition = (
  targetElement: HTMLElement,
  hintDialogDiv: HTMLElement,
  position: DialogPosition = "bottom"
) => {
  const bounding = getElementBounding(targetElement);
  const hintDialogBounding = getElementBounding(hintDialogDiv);

  if (position === "bottom") {
    hintDialogDiv.style.top = `${bounding.top + bounding.height}px`;
    hintDialogDiv.style.left = `${bounding.left - 4}px`;
  } else if (position === "top") {
    hintDialogDiv.style.top = `${
      bounding.top - hintDialogBounding.height - 8
    }px`;
    hintDialogDiv.style.left = `${bounding.left - 4}px`;
  } else if (position === "left") {
    hintDialogDiv.style.top = `${bounding.top - 4}px`;
    hintDialogDiv.style.left = `${
      bounding.left - hintDialogBounding.width - 8
    }px`;
  } else if (position === "right") {
    hintDialogDiv.style.top = `${bounding.top - 4}px`;
    hintDialogDiv.style.left = `${bounding.left + bounding.width}px`;
  } else if (position === "topright") {
    hintDialogDiv.style.top = `${
      bounding.top - hintDialogBounding.height - 8
    }px`;
    hintDialogDiv.style.left = `${
      bounding.left + bounding.width - hintDialogBounding.width
    }px`;
  }

  requestAnimationFrame(() =>
    adjustHintDialogPosition(targetElement, hintDialogDiv, position)
  );
};

const updateHintPosition = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  const hintWrapper = document.body.querySelector(
    `.bb-hint-wrapper.bb-hint-${hintIndex}`
  ) as HTMLElement;

  if (!targetElement || !hintWrapper) {
    removeHint(hintIndex);
    return;
  }

  const bounding = getElementBounding(targetElement);

  hintWrapper.style.top = `${bounding.top}px`;
  hintWrapper.style.left = `${bounding.left}px`;
  hintWrapper.style.width = `${bounding.width}px`;
  hintWrapper.style.height = `${bounding.height}px`;

  requestAnimationFrame(() =>
    updateHintPosition(targetElement, hintData, hintIndex)
  );
};

export const removeHint = (hintIndex?: number) => {
  if (isUndefined(hintIndex)) {
    const hintWrappers = document.querySelectorAll(".bb-hint-wrapper");
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
    const hintCovers = document.querySelectorAll(".bb-hint-cover-wrapper");
    if (hintCovers) {
      hintCovers.forEach((item) => {
        item.remove();
      });
    }
  } else {
    const hintWrapper = document.querySelector(
      `.bb-hint-wrapper.bb-hint-${hintIndex}`
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

    const hintCover = document.querySelector(
      `.bb-hint-cover-wrapper.bb-hint-${hintIndex}`
    );
    if (hintCover) {
      hintCover.remove();
    }
  }
};
