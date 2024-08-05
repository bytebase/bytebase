export type OAuthWindowEventPayload = {
  error: string;
  code: string;
};

export type OAuthState = {
  event: string;
  popup?: boolean;
  redirect?: string;
};
