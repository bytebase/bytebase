import { v4 as uuidv4 } from "uuid";
import { DataSource, DataSourceType } from "../proto/v1/instance_service";

export const emptyDataSource = () => {
  return DataSource.fromJSON({
    type: DataSourceType.ADMIN,
    id: uuidv4(),
  });
};

export const unknownDataSource = () => {
  return {
    ...emptyDataSource(),
    title: "<<Unknown data source>>",
  };
};
