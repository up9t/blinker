export const NotificationEvent = {
  SEND: "notification:send",
};

export interface INotificationService {
  requestPermission(): Promise<boolean>;
  showNotification(title: string, body: string): Promise<void>;
}
