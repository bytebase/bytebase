export type BBTableColumn = {
  title: string;
};

export type BBTableSectionDataSource<T> = {
  title: string;
  list: T[];
};

export type BBStepStatus =
  | "CREATED"
  | "PENDING"
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
