import { PipelineId } from "../id";
import { Stage, StageCreate } from "./stage";

/*
 An example

 An alter schema PIPELINE
  Dev STAGE (db_dev, env_dev)
    Change dev database schema

  Testing STAGE (db_test, env_test)
    Change testing database schema
    Verify integration test pass

  Staging STAGE (db_staging, env_staging)
    Approve change
    Change staging database schema

  Prod STAGE (db_prod, env_prod)
    Approve change
    Change prod database schema
*/
// Pipeline
export type PipelineStatus = "OPEN" | "DONE" | "CANCELED";

export type Pipeline = {
  id: PipelineId;

  // Related fields
  stageList: Stage[];

  // Domain specific fields
  name: string;
};

export type PipelineCreate = {
  // Related fields
  stageList: StageCreate[];

  // Domain specific fields
  name: string;
};

export type PipelineStatusPatch = {
  // Domain specific fields
  status: PipelineStatus;
  comment?: string;
};
