import { BookmarkId, PrincipalId } from "./id";
import { Principal } from "./principal";

export type Bookmark = {
  id: BookmarkId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  link: string;
};

export type BookmarkNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  link: string;
};
