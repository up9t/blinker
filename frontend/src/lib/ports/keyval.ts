export interface IKeyValueService<K, V extends any> {
  getValue(key: K): Promise<V>;
  setValue(key: K, value: V): Promise<void>;
}
