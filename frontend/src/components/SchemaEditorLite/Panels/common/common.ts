import { v1 as uuidv1 } from "uuid";
import type { EditStatus } from "../../types";

export const markUUID = (obj: any) => {
  // column.name is editable, so we need to insert another hidden field
  // as a column's stable unique key.
  if (!obj.__uuid) {
    // Make this field 'invisible' to avoid it breaks lodash's deep comparing
    Object.defineProperty(obj, "__uuid", {
      enumerable: false,
      writable: false,
      configurable: false,
      value: uuidv1(),
    });
  }
  return obj.__uuid as string;
};

export const markEditStatusBeforeDrop = (obj: any, status: EditStatus) => {
  Object.defineProperty(obj, "__status_before_drop", {
    enumerable: false,
    writable: true,
    configurable: false,
    value: status,
  });
};

export const getEditStatusBeforeDrop = (obj: any) => {
  return obj["__status_before_drop"] as EditStatus | undefined;
};
