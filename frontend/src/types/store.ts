import type { Command } from "./common";
import type { CommandId } from "./id";
import type { Notification } from "./notification";

export interface NotificationState {
  notificationByModule: Map<string, Notification[]>;
}

export interface CommandState {
  commandListById: Map<CommandId, Command[]>;
}
