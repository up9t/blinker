import {
  CheckNotificationAuthorization,
  RequestNotificationAuthorization,
} from "../../../bindings/github.com/wailsapp/wails/v3/pkg/services/notifications/notificationservice.js";
import { NotificationEvent, type INotificationService } from "../ports/notification.ts";
import { Events } from "@wailsio/runtime";

export class WailsNotificationService implements INotificationService {
  async requestPermission(): Promise<boolean> {
    const authorized = await CheckNotificationAuthorization();

    if (!authorized) {
      return await RequestNotificationAuthorization();
    }

    return true;
  }

  async showNotification(title: string, body: string): Promise<void> {
    const data = {
      title,
      body,
    } satisfies {
      title: string;
      body: string;
    };

    await Events.Emit(NotificationEvent.SEND, data);
  }
}
