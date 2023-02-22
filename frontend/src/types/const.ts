// unknown represents an anomaly.
// Returns as function to avoid caller accidentally mutate it.
// UNKNOWN_ID means an anomaly, it expects a resource which is missing (e.g. Keyed lookup missing).
export const UNKNOWN_ID = -1;
// EMPTY_ID means an expected behavior, it expects no resource (e.g. contains an empty value, using this technic enables
// us to declare variable as required, which leads to cleaner code)
export const EMPTY_ID = 0;

export const UNKNOWN_NAME = "<<Unknown>>";

export const EMPTY_NAME = "<<Empty>>";

export const UNKNOWN_UID = "";
