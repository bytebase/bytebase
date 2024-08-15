export type EditorPanelView =
  | "CODE"
  | "INFO"
  | "TABLES"
  | "VIEWS"
  | "FUNCTIONS"
  | "PROCEDURES"
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
  throw new Error(`unsupported type: "${type}"`);
};
