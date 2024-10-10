export interface DatabaseSelectState {
  changeSource: "DATABASE" | "GROUP";
  selectedDatabaseNameList: string[];
  selectedDatabaseGroup?: string;
}
