import { BookmarkId, PrincipalId } from "./id";

export type Bookmark = {
  id: BookmarkId;

  // Standard fields
  creatorID: PrincipalId;

  // Domain specific fields
  name: string;
  link: string;
};

export type BookmarkCreate = {
  // Domain specific fields
  name: string;
  link: string;
};
