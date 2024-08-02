export type EditorPanelView =
  | "CODE"
  | "INFO"
  | "TABLES"
  | "VIEWS"
  | "FUNCTIONS"
  | "PROCEDURES";

export type EditorPanelViewState = {
  view: EditorPanelView;
};

export const defaultViewState = (): EditorPanelViewState => {
  return {
    view: "CODE",
  };
};
