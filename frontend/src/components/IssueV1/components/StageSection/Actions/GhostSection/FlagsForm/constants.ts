export type GhostParameterType = "bool" | "int" | "float" | "string";

export type GhostParameter<T extends GhostParameterType = any> = {
  key: string;
  type: T;
};

export type GhostParameterWithValue<T extends GhostParameterType = any> =
  GhostParameter<T> & {
    value: string;
  };

export const SupportedGhostParameters: GhostParameter[] = [
  { key: "max-load", type: "string" },
  { key: "chunk-size", type: "int" },
  { key: "initially-drop-ghost-table", type: "bool" },
  { key: "max-lag-millis", type: "int" },
  { key: "allow-on-master", type: "bool" },
  { key: "switch-to-rbr", type: "bool" },
];

export const DefaultGhostParameters: GhostParameterWithValue[] = [
  {
    key: "allow-on-master",
    type: "bool",
    value: "true",
  },
  {
    key: "concurrent-rowcount",
    type: "bool",
    value: "true",
  },
  {
    key: "timestampAllTable",
    type: "bool",
    value: "true",
  },
  {
    key: "hooks-status-interval",
    type: "int",
    value: "60",
  },
  {
    key: "heartbeat-interval-millis",
    type: "int",
    value: "100",
  },
  {
    key: "nice-ratio",
    type: "float",
    value: "0",
  },
  {
    key: "chunk-size",
    type: "int",
    value: "1000",
  },
  {
    key: "dml-batch-size",
    type: "int",
    value: "10",
  },
  {
    key: "max-lag-millis",
    type: "int",
    value: "1500",
  },
  {
    key: "default-retries",
    type: "int",
    value: "60",
  },
  {
    key: "cut-over-lock-timeout-seconds",
    type: "int",
    value: "60",
  },
  {
    key: "exponential-backoff-max-interval",
    type: "int",
    value: "64",
  },
  {
    key: "throttle-http-interval-millis",
    type: "int",
    value: "100",
  },
  {
    key: "throttle-http-timeout-millis",
    type: "int",
    value: "1000",
  },
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
