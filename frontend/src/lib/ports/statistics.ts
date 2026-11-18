// mirrors StatBucket in statistics.go
export interface IStatBucket {
  // unix start of the local hour (today) or local day (week, month)
  start: number;
  value: number;
}

export interface IStatPeriod {
  total: number;
  // null when the backend sends no buckets
  buckets: IStatBucket[] | null;
}

// mirrors StatPeriodSet in statistics.go
export interface IStatPeriodSet {
  today: IStatPeriod;
  week: IStatPeriod;
  month: IStatPeriod;
}

// mirrors Statistics in statistics.go
export interface IStatistics {
  blinks: IStatPeriodSet;
  // values are millis
  screenTime: IStatPeriodSet;
}

export interface IStatisticsService {
  statistics(): Promise<IStatistics>;
}
