export type MessageAuthor = "USER" | "AI";

export type MessageStatus = "LOADING" | "DONE" | "FAILED";

export type Message = {
  id: string; // uuid
  created_ts: number; // ms
  author: MessageAuthor;
  content: string; // Content to display
  prompt: string; // Prompt to send to AI
  status: MessageStatus; // Always "DONE" when `author`="USER"
  error: string;
  conversation: Conversation;
};

export type Conversation = {
  id: string; // uuid
  created_ts: number; // ms
  name: string;
  messageList: Message[];
};
