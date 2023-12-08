import { v1 as uuidv1 } from "uuid";

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
