import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Worksheet as OldWorksheet, WorksheetOrganizer as OldWorksheetOrganizer } from "@/types/proto/v1/worksheet_service";
import { Worksheet as OldWorksheetProto, WorksheetOrganizer as OldWorksheetOrganizerProto, Worksheet_Visibility as OldVisibility } from "@/types/proto/v1/worksheet_service";
import type { Worksheet as NewWorksheet, WorksheetOrganizer as NewWorksheetOrganizer } from "@/types/proto-es/v1/worksheet_service_pb";
import { WorksheetSchema, WorksheetOrganizerSchema, Worksheet_Visibility as NewVisibility } from "@/types/proto-es/v1/worksheet_service_pb";

// Convert old proto to proto-es
export const convertOldWorksheetToNew = (oldWorksheet: OldWorksheet): NewWorksheet => {
  const json = OldWorksheetProto.toJSON(oldWorksheet) as any;
  return fromJson(WorksheetSchema, json);
};

// Convert proto-es to old proto
export const convertNewWorksheetToOld = (newWorksheet: NewWorksheet): OldWorksheet => {
  const json = toJson(WorksheetSchema, newWorksheet);
  return OldWorksheetProto.fromJSON(json);
};

// Convert old WorksheetOrganizer to new
export const convertOldWorksheetOrganizerToNew = (oldOrganizer: OldWorksheetOrganizer): NewWorksheetOrganizer => {
  const json = OldWorksheetOrganizerProto.toJSON(oldOrganizer) as any;
  return fromJson(WorksheetOrganizerSchema, json);
};

// Convert new WorksheetOrganizer to old
export const convertNewWorksheetOrganizerToOld = (newOrganizer: NewWorksheetOrganizer): OldWorksheetOrganizer => {
  const json = toJson(WorksheetOrganizerSchema, newOrganizer);
  return OldWorksheetOrganizerProto.fromJSON(json);
};

// Convert visibility enum
export const convertOldVisibilityToNew = (oldVisibility: OldVisibility): NewVisibility => {
  const mapping: Record<OldVisibility, NewVisibility> = {
    [OldVisibility.VISIBILITY_UNSPECIFIED]: NewVisibility.UNSPECIFIED,
    [OldVisibility.VISIBILITY_PROJECT_READ]: NewVisibility.PROJECT_READ,
    [OldVisibility.VISIBILITY_PROJECT_WRITE]: NewVisibility.PROJECT_WRITE,
    [OldVisibility.VISIBILITY_PRIVATE]: NewVisibility.PRIVATE,
    [OldVisibility.UNRECOGNIZED]: NewVisibility.UNSPECIFIED,
  };
  return mapping[oldVisibility] ?? NewVisibility.UNSPECIFIED;
};

export const convertNewVisibilityToOld = (newVisibility: NewVisibility): OldVisibility => {
  const mapping: Record<NewVisibility, OldVisibility> = {
    [NewVisibility.UNSPECIFIED]: OldVisibility.VISIBILITY_UNSPECIFIED,
    [NewVisibility.PROJECT_READ]: OldVisibility.VISIBILITY_PROJECT_READ,
    [NewVisibility.PROJECT_WRITE]: OldVisibility.VISIBILITY_PROJECT_WRITE,
    [NewVisibility.PRIVATE]: OldVisibility.VISIBILITY_PRIVATE,
  };
  return mapping[newVisibility] ?? OldVisibility.UNRECOGNIZED;
};