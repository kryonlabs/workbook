// Native document/profile state feeds the TSX editor and menu composition.
import * as App from "kryon/app";
import * as Host from "app";

export function Frame(frame: App.Frame): void {
    Host.DesktopFrame(frame);
}

export function main(): s32 {
    return App.Run({title: "Workbook"}, Frame);
}
