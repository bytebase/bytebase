import { QueryHistory } from ".";
import { Command } from "./common";
import { CommandId } from "./id";
import { Notification } from "./notification";

export interface NotificationState {
  notificationByModule: Map<string, Notification[]>;
}

export interface CommandState {
  commandListById: Map<CommandId, Command[]>;
}

export interface SQLEditorState {
  shouldFormatContent: boolean;
  queryHistoryList: QueryHistory[];
  isFetchingQueryHistory: boolean;
  isFetchingSheet: boolean;
  isShowExecutingHint: boolean;
}
