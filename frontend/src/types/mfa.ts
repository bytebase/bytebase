export interface MFAChallengeContext {
  otpCode?: string;
  recoveryCode?: string;
}

export type MFAChallengeCallback = (context: MFAChallengeContext) => any;
