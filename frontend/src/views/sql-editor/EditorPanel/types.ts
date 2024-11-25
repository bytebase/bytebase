export type EditorPanelView =
  | "CODE"
  | "INFO"
  | "TABLES"
  | "VIEWS"
  | "FUNCTIONS"
  | "PROCEDURES"
  | "SEQUENCES"
  | "TRIGGERS"
  | "EXTERNAL_TABLES"
  | "PACKAGES"
  | "DIAGRAM";

export type EditorPanelViewState = {
  view: EditorPanelView;
  schema?: string;
  detail: {
    table?: string;
    column?: string;
    view?: string;
    procedure?: string;
    func?: string;
    sequence?: string;
    trigger?: string;
    externalTable?: string;
    partition?: string;
    index?: string;
    foreignKey?: string;
    dependentColumn?: string;
    package?: string;
  };
};

export const defaultViewState = (): EditorPanelViewState => {
  return {
    view: "CODE",
    schema: undefined,
    detail: {},
  };
};

export const typeToView = (type: string): EditorPanelView => {
  if (
    type === "table" ||
    type === "column" ||
    type === "index" ||
    type === "foreign-key" ||
    type === "partition-table"
  ) {
    return "TABLES";
  }
  if (type === "view" || type === "dependent-column") {
    return "VIEWS";
  }
  if (type === "function") return "FUNCTIONS";
  if (type === "procedure") return "PROCEDURES";
  if (type === "external-table") return "EXTERNAL_TABLES";
  if (type === "package") return "PACKAGES";
  if (type === "sequence") return "SEQUENCES";
  if (type === "trigger") return "TRIGGERS";
  throw new Error(`unsupported type: "${type}"`);
};
