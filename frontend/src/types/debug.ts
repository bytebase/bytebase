export type Debug = {
  isDebug: boolean;
};

export type DebugPatch = {
  isDebug: boolean;
};

export type DebugLog = {
  RecordTs: number;
  Method: string;
  RequestPath: string;
  Role: string;
  Error: string;
  StackTrace: string;
};
