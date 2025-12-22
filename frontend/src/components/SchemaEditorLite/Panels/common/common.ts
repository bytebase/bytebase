import { v1 as uuidv1 } from "uuid";
import type { EditStatus } from "../../types";

// Hidden properties added to schema objects
interface HiddenProps {
  __uuid?: string;
  __status_before_drop?: EditStatus;
}

type ObjectWithHiddenProps = object & HiddenProps;

export const markUUID = (obj: object): string => {
  // column.name is editable, so we need to insert another hidden field
  // as a column's stable unique key.
  const target = obj as ObjectWithHiddenProps;
  if (!target.__uuid) {
    // Make this field 'invisible' to avoid it breaks lodash's deep comparing
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
  Object.defineProperty(obj, "__status_before_drop", {
    enumerable: false,
    writable: true,
    configurable: false,
    value: status,
  });
};

export const getEditStatusBeforeDrop = (
  obj: object
): EditStatus | undefined => {
  return (obj as ObjectWithHiddenProps).__status_before_drop;
};
