import { IssueID } from "./id";
import { Principal } from "./principal";

export type IssueSubscriber = {
  // Domain specific fields
  issueID: IssueID;
  subscriber: Principal;
};
