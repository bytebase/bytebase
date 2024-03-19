import { Command } from "./common";
import { CommandId } from "./id";
import { Notification } from "./notification";

export interface NotificationState {
  notificationByModule: Map<string, Notification[]>;
}

export interface CommandState {
  commandListById: Map<CommandId, Command[]>;
}
