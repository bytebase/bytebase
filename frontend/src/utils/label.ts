import { Label } from "../types";

export const RESERVED_LABEL_ID = 0;

export const isReservedLabel = (label: Label): boolean => {
  return label.id === RESERVED_LABEL_ID;
};
