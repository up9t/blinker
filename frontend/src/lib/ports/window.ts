export const WindowsName = {
  MAIN: "main",
  OVERLAY: "overlay",
};

// export const WindowsEvent = {
//   HIDE: "window:hide",
//   SHOW: "window:show",
//   FOCUS: "window:focus",
//   SHOW_AND_FOCUS: "window:show_and_focus"
// }

export interface IWindowService {
  hideWindow(name: (typeof WindowsName)[keyof typeof WindowsName]): Promise<void>;
  showWindow(name: (typeof WindowsName)[keyof typeof WindowsName], focus: boolean): Promise<void>;
}
