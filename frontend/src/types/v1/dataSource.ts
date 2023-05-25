import { DataSource, DataSourceType } from "../proto/v1/instance_service";

export const emptyDataSource = () => {
  return DataSource.fromJSON({
    type: DataSourceType.ADMIN,
  });
};

export const unknownDataSource = () => {
  return {
    ...emptyDataSource(),
    title: "<<Unknown data source>>",
  };
};
