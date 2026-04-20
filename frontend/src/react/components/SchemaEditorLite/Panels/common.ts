import { v1 as uuidv1 } from "uuid";
import type { EditStatus } from "../types";

interface ObjectWithHiddenProps {
  __uuid?: string;
  __status_before_drop?: EditStatus;
}

export const markUUID = (obj: object): string => {
  const target = obj as ObjectWithHiddenProps;
  if (!target.__uuid) {
    Object.defineProperty(obj, "__uuid", {
      enumerable: false,
      writable: false,
      configurable: false,
      value: uuidv1(),
    });
  }
  return target.__uuid as string;
};

export const markEditStatusBeforeDrop = (obj: object, status: EditStatus) => {
  const target = obj as ObjectWithHiddenProps;
  Object.defineProperty(target, "__status_before_drop", {
    enumerable: false,
    writable: true,
    configurable: true,
    value: status,
  });
};

export const getEditStatusBeforeDrop = (
  obj: object
): EditStatus | undefined => {
  return (obj as ObjectWithHiddenProps).__status_before_drop;
};
