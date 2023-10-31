export type GhostParameterType = "bool" | "int" | "string";

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
