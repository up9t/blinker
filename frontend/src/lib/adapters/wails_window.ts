import { type IWindowService, type WindowsName } from "../ports/window.ts";
import { Window } from "@wailsio/runtime";

export class WailsWindowService implements IWindowService {
  async hideWindow(name: (typeof WindowsName)[keyof typeof WindowsName]): Promise<void> {
    await Window.Get(name).Hide();
  }

  async showWindow(name: (typeof WindowsName)[keyof typeof WindowsName], focus: boolean = true): Promise<void> {
    const window = Window.Get(name);

    await window.Show().then(() => {
      if (focus) {
        window.Focus();
      }
    });
  }
}
