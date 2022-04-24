import { SheetId } from ".";

export type SheetOrganizerUpsert = {
  sheeId: SheetId;
  starred?: boolean;
  pinned?: boolean;
};
