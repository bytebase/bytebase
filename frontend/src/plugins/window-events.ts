export type WindowEventActivityType =
  | "bb.issue-create"
  | "bb.issue-detail"
  | "bb.issue-field-update"
  | "bb.pipeline-task-statement-update"
  | "bb.pipeline-task-earliest-allowed-time-update";

export const emitWindowEvent = (
  activity: WindowEventActivityType,
  params: any = undefined
) => {
  const data: Record<string, any> = { activity };
  if (params) {
    data.params = params;
  }
  window.parent.postMessage(data, "*");

  if (window.opener && typeof window.opener.postMessage === "function") {
    window.opener.postMessage(data, "*");
  }
};
