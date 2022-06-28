import { assign } from "lodash-es";

export const getStylePropertyValue = (
  element: HTMLElement,
  propertyName: string
) => {
  const propertyValue = window
    .getComputedStyle(element)
    .getPropertyValue(propertyName);

  return propertyValue;
};

export const isElementFixed = (element: HTMLElement): boolean => {
  const parentNode = element.parentNode;

  if (!parentNode || parentNode.nodeName === "HTML") {
    return false;
  }

  if (getStylePropertyValue(element, "position") === "fixed") {
    return true;
  }

  return isElementFixed(parentNode as HTMLElement);
};

export const getElementBounding = (
  element: HTMLElement,
  relativeEl?: HTMLElement
) => {
  const scrollTop =
    window.pageYOffset ||
    document.documentElement.scrollTop ||
    document.body.scrollTop;
  const scrollLeft =
    window.pageXOffset ||
    document.documentElement.scrollLeft ||
    document.body.scrollLeft;

  relativeEl = relativeEl || document.body;

  const elementRect = element.getBoundingClientRect();
  const relativeElRect = relativeEl.getBoundingClientRect();
  const relativeElPosition = getStylePropertyValue(relativeEl, "position");

  const bounding = {
    width: elementRect.width,
    height: elementRect.height,
  };

  if (
    (relativeEl.tagName !== "BODY" && relativeElPosition === "relative") ||
    relativeElPosition === "sticky"
  ) {
    return assign(bounding, {
      top: elementRect.top - relativeElRect.top,
      left: elementRect.left - relativeElRect.left,
    });
  }

  if (isElementFixed(element)) {
    return assign(bounding, {
      top: elementRect.top,
      left: elementRect.left,
    });
  }

  return assign(bounding, {
    top: elementRect.top + scrollTop,
    left: elementRect.left + scrollLeft,
  });
};
