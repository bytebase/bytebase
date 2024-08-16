export type EditorPanelView =
  | "CODE"
  | "INFO"
  | "TABLES"
  | "VIEWS"
  | "FUNCTIONS"
  | "PROCEDURES"
  | "EXTERNAL_TABLES"
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
    externalTable?: string;
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
  if (type === "table" || type === "column") return "TABLES";
  if (type === "view") return "VIEWS";
  if (type === "function") return "FUNCTIONS";
  if (type === "procedure") return "PROCEDURES";
  if (type === "external-table") return "EXTERNAL_TABLES";
  throw new Error(`unsupported type: "${type}"`);
};
