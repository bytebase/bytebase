import { Label } from "../types";

// reserved labels (e.g. bb.environment) have zero ID and their values are immutable.
// see api/label.go for more details
export const RESERVED_LABEL_ID = 0;

export const isReservedLabel = (label: Label): boolean => {
  return label.id === RESERVED_LABEL_ID;
};
