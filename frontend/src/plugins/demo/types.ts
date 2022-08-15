export interface ProcessData {
  url: string;
  title: string | I18NText;
  description: string | I18NText;
}

export interface DemoData {
  process: ProcessData[];
  hint: HintData[];
}

type StepType = "click" | "change";

export type Position =
  | "bottom"
  | "top"
  | "left"
  | "right"
  | "topright"
  | "center"
  | "leftcenter";

export type I18NText = {
  [key: string]: string;
};

export interface StepData {
  type: StepType;
  title: string | I18NText;
  description: string | I18NText;
  selectors: string[][];
  // url is using for validate url in change step
  url?: string;
  // value is the regex-like string using for check the target content value
  value?: string;
  // position is the position of the guide dialog (default is bottom)
  position?: Position;
  // cover is the flag that cover should be shown
  cover?: boolean;
  // hideNextButton is the flag that next button should be hidden
  hideNextButton?: boolean;
}

export interface GuideData {
  name: string;
  steps: StepData[];
}

export type HintType = "tooltip" | "shield";

// Hint is a special guide that has no Next button and is always shown.
export interface HintData {
  selector: string;
  type: HintType;
  // pathname is the wanted pathname of the url, cound be a regex-like string
  pathname: string;
  // url is using for validate url in change step
  url: string;
  highlight?: boolean;
  // cover is the flag that cover should be shown
  cover?: boolean;
  // position is the position of the hint espectially for tooltip (default is right)
  position?: Position;
  // addStyle for customizing the actual style
  // dialog is a data of dialog info. If it's undefined, then the dialog will not be shown.
  dialog?: {
    title: string | I18NText;
    description: string | I18NText;
    // position is the position of the guide dialog (default is bottom)
    position?: Position;
    alwaysShow?: boolean;
    showOnce?: boolean;
  };
  additionStyle?: CSSStyleDeclaration;
}
