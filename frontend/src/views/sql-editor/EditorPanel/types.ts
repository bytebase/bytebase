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
};

export const defaultViewState = (): EditorPanelViewState => {
  return {
    view: "CODE",
    schema: undefined,
  };
};
