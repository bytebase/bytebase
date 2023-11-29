export type GhostParameterType = "bool" | "int" | "float" | "string";

export type GhostParameter<T extends GhostParameterType = any> = {
  key: string;
  type: T;
  defaults: string;
};

export const SupportedGhostParameters: GhostParameter[] = [
  { key: "max-load", type: "string", defaults: "" },
  { key: "chunk-size", type: "int", defaults: "1000" },
  { key: "dml-batch-size", type: "int", defaults: "10" },
  { key: "default-retries", type: "int", defaults: "60" },
  { key: "cut-over-lock-timeout-seconds", type: "int", defaults: "10" },
  { key: "exponential-backoff-max-interval", type: "int", defaults: "64" },
  { key: "max-lag-millis", type: "int", defaults: "1500" },
  { key: "allow-on-master", type: "bool", defaults: "true" },
  { key: "switch-to-rbr", type: "bool", defaults: "false" },
  { key: "assume-rbr", type: "bool", defaults: "false" },
  { key: "heart-beat-interval-millis", type: "int", defaults: "100" },
  { key: "nice-ratio", type: "float", defaults: "0" },
  { key: "aliyun-rds", type: "bool", defaults: "false" },
  { key: "azure", type: "bool", defaults: "false" },
  { key: "gcp", type: "bool", defaults: "false" },
];

export const isBoolParameter = (
  param: GhostParameter
): param is GhostParameter<"bool"> => {
  return param.type === "bool";
};

export const isIntParameter = (
  param: GhostParameter
): param is GhostParameter<"int"> => {
  return param.type === "int";
};

export const isStringParameter = (
  param: GhostParameter
): param is GhostParameter<"string"> => {
  return param.type === "string";
};

export const isFloatParameter = (
  param: GhostParameter
): param is GhostParameter<"float"> => {
  return param.type === "float";
};
