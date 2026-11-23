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

// mirrors TodayStatistics in statistics.go, screenTime in millis
export interface ITodayStatistics {
  blinks: number;
  screenTime: number;
}

export interface IStatisticsService {
  statistics(): Promise<IStatistics>;
  // one total per metric for today, no buckets
  today(): Promise<ITodayStatistics>;
}
