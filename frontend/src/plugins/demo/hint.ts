import { removeElementBySelector } from "@/utils";
import { indexOf, isUndefined } from "lodash-es";
import { getStringFromI18NText } from "./i18n";
import { Position, HintData } from "./types";
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
    if (
      shownHintIndexSet.has(index) ||
      closedHintIndexSet.has(index) ||
      !checkUrlMatched(hintData.url)
    ) {
      return;
    }

    const targetElement = await waitForTargetElement([[hintData.selector]]);
    if (targetElement) {
      renderHint(targetElement, hintData, index);
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
  if (hintData.type === "tooltip") {
    renderTooltip(targetElement, hintData, hintIndex);
    return;
  }

  const hintWrapper = document.createElement("div");
  hintWrapper.className = `bb-hint-wrapper bb-hint-${hintIndex}`;
  if (hintData.type === "shield") {
    hintWrapper.classList.add("shield");
  }
  removeElementBySelector(`.bb-hint-wrapper.bb-hint-${hintIndex}`);
  document.body.appendChild(hintWrapper);

  if (hintData.cover) {
    hintWrapper.classList.add("covered");
    renderCover(targetElement, hintIndex);
  }

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
  hintWrapper.style.pointerEvents = "none";

  if (hintData.dialog) {
    if (hintData.dialog.alwaysShow) {
      renderHintDialog(targetElement, hintData, hintIndex);
    } else {
      hintWrapper.style.pointerEvents = "auto";
      hintWrapper.addEventListener(
        "click",
        (e) => {
          e.stopPropagation();
          hintWrapper.style.pointerEvents = "none";
          renderHintDialog(targetElement, hintData, hintIndex);
          if (hintData.dialog?.showOnce) {
            closedHintIndexSet.add(hintIndex);
          }
        },
        {
          once: true,
        }
      );
    }
  }

  updateHintPosition(targetElement, hintData, hintIndex);
};

const renderHintDialog = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  if (isUndefined(hintData.dialog) || closedHintIndexSet.has(hintIndex)) {
    return;
  }

  const {
    dialog: { title, description, position, alwaysShow },
  } = hintData;
  const hintDialogDiv = document.createElement("div");
  hintDialogDiv.className = `bb-hint-dialog bb-hint-${hintIndex} ${
    position ?? "bottom"
  }`;

  if (hintData.type === "tooltip") {
    hintDialogDiv.classList.add("tooltip-dialog");
  }

  const maxZIndex = getElementMaxZIndex(targetElement);
  hintDialogDiv.style.zIndex = `${maxZIndex}`;

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

  removeElementBySelector(`.bb-hint-dialog.bb-hint-${hintIndex}`);
  document.body.appendChild(hintDialogDiv);
  adjustHintDialogPosition(targetElement, hintDialogDiv, position);

  return hintDialogDiv;
};

const adjustHintDialogPosition = (
  targetElement: HTMLElement,
  hintDialogDiv: HTMLElement,
  position: Position = "bottom"
) => {
  const bounding = getElementBounding(targetElement);
  const hintDialogBounding = getElementBounding(hintDialogDiv);

  if (bounding.width === 0 && bounding.height === 0) {
    hintDialogDiv.remove();
    return;
  }

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

const renderTooltip = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  const tooltipWrapper = document.createElement("div");
  tooltipWrapper.className = `bb-hint-tooltip-wrapper bb-hint-${hintIndex}`;
  const pingElement = document.createElement("span");
  pingElement.className = "bb-hint-tooltip-ping";
  tooltipWrapper.appendChild(pingElement);

  tooltipWrapper.style.zIndex = `${getElementMaxZIndex(targetElement)}`;
  if (hintData.additionStyle) {
    for (const key in hintData.additionStyle) {
      tooltipWrapper.style[key] = `${hintData.additionStyle[key]}`;
    }
  }
  removeElementBySelector(`.bb-hint-tooltip-wrapper.bb-hint-${hintIndex}`);
  document.body.appendChild(tooltipWrapper);

  tooltipWrapper.addEventListener("mouseenter", () => {
    const dialogElement = renderHintDialog(tooltipWrapper, hintData, hintIndex);
    tooltipWrapper.addEventListener(
      "mouseleave",
      () => {
        dialogElement?.remove();
      },
      {
        once: true,
      }
    );
  });

  tooltipWrapper.addEventListener("click", () => {
    closedHintIndexSet.add(hintIndex);
    removeHint(hintIndex);
  });

  requestAnimationFrame(() =>
    updateTooltipPosition(targetElement, hintData, hintIndex)
  );
};

const updateTooltipPosition = (
  targetElement: HTMLElement,
  hintData: HintData,
  hintIndex: number
) => {
  const hintWrapper = document.body.querySelector(
    `.bb-hint-tooltip-wrapper.bb-hint-${hintIndex}`
  ) as HTMLElement;

  if (!hintWrapper) {
    removeHint(hintIndex);
    return;
  }

  const position = hintData.position;
  const bounding = getElementBounding(targetElement);

  if (bounding.width === 0 && bounding.height === 0) {
    removeHint(hintIndex);
    return;
  }

  if (!position || position === "right") {
    hintWrapper.style.top = `${bounding.top}px`;
    hintWrapper.style.left = `${bounding.left + bounding.width}px`;
  } else if (position === "left") {
    hintWrapper.style.top = `${bounding.top}px`;
    hintWrapper.style.left = `${bounding.left}px`;
  }

  requestAnimationFrame(() =>
    updateTooltipPosition(targetElement, hintData, hintIndex)
  );
};

const renderCover = (targetElement: HTMLElement, hintIndex: number) => {
  const coverElement = document.createElement("div");
  coverElement.className = `bb-hint-cover-wrapper bb-hint-${hintIndex}`;
  const maxZIndex = getElementMaxZIndex(targetElement);
  coverElement.style.zIndex = `${Math.max(maxZIndex - 1, 0)}`;
  removeElementBySelector(`.bb-hint-cover-wrapper.bb-hint-${hintIndex}`);
  document.body.appendChild(coverElement);
  targetElement.classList.add("bb-hint-target-element");
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

    const hintTooltips = document.querySelectorAll(".bb-hint-tooltip-wrapper");
    if (hintTooltips) {
      hintTooltips.forEach((item) => {
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
      `.bb-hint-dialog.bb-hint-${hintIndex}`
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

    const hintTooltip = document.querySelector(
      `.bb-hint-tooltip-wrapper.bb-hint-${hintIndex}`
    );
    if (hintTooltip) {
      hintTooltip.remove();
    }
  }
};
