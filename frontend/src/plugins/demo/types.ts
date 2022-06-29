type StepType = "click" | "change";

export type GuidePosition = "bottom" | "top" | "left" | "right";

export interface StepData {
  type: StepType;
  title: string;
  description: string;
  selectors: string[][];
  // url is using for validate url in change step
  url?: string;
  // value is the regex-like string using for check the target content value
  value?: string;
  // position is the position of the guide dialog, default is bottom
  position?: GuidePosition;
}

export interface GuideData {
  name: string;
  steps: StepData[];
  // cover is the flag that should be shown for the guide dialog
  cover?: boolean;
}
