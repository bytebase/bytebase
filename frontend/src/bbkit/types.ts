export type BBButtonType =
  | "NORMAL"
  | "PRIMARY"
  | "SECONDARY"
  | "DANGER"
  | "SUCCESS";

export type BBButtonConfirmType = "NORMAL" | "DELETE" | "ARCHIVE" | "RESTORE";

export type BBTabItem<T = unknown> = {
  title: string;
  // Used as the anchor
  id: string;
  data?: T;
};

export type BBAvatarSizeType =
  | "MINI"
  | "TINY"
  | "SMALL"
  | "NORMAL"
  | "LARGE"
  | "HUGE";
