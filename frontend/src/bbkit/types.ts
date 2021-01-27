export type BBTableColumn = {
  title: string;
};

export type BBStepStatus =
  | "CREATED"
  | "RUNNING"
  | "DONE"
  | "FAILED"
  | "CANCELED"
  | "SKIPPED";

export type BBStep = {
  title: string;
  status: BBStepStatus;
  link(): string;
};
