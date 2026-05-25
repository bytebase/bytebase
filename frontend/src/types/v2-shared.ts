export type ResourceSelectOption<T> = {
  resource?: T;
  value: string;
  label: string;
  // Allow arbitrary extra fields the Vue-side select components may set;
  // React consumers only read value/label/resource.
  [key: string]: unknown;
};

export type SelectSize = "tiny" | "small" | "medium" | "large";
