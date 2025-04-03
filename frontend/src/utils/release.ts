import type { Release_File } from "@/types/proto/v1/release_service";

export const getReleaseFileStatement = (file: Release_File) => {
  return new TextDecoder().decode(file.statement);
};
