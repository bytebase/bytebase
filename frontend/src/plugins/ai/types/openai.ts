export type OpenAIMessage = {
  role: "system" | "user" | "assistant";
  content: string;
};

export type OpenAIChoice = {
  // text: string;
  index: number;
  logprobs: unknown;
  finish_reason: "stop" | "length" | null;
  message: OpenAIMessage;
};

export type OpenAIUsage = {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
};

export type OpenAIError = {
  message: string;
  type: string;
  param: unknown;
  code: unknown;
};

export type OpenAIResponse = {
  id: string;
  object: "text_completion";
  created: number;
  model: "code-davinci-002";
  choices: OpenAIChoice[];
  usage: OpenAIUsage;
  error: OpenAIError | undefined;
};
