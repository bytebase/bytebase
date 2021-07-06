import { IssueId } from "./id";
import { Principal } from "./principal";

export type IssueSubscriber = {
  // Domain specific fields
  issueId: IssueId;
  subscriber: Principal;
};
