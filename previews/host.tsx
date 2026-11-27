// A real TSX host around the public embedded editor, driven on a private display.
import * as App from "kryon/app";
import * as State from "host_state";
import * as Editor from "workbook/editor";
import * as Grid from "grid";
import * as NativeText from "kryon/text";
import { Box } from "kryon/tsx/box";
import { Button } from "kryon/tsx/button";
import { Text } from "kryon/tsx/text";
import { Rectangle } from "kryon/geometry";
import { Session } from "kryon/session";

function ActionLabel(id: s32): string {
    if (id === s32(1)) { return "Host focus"; }
    if (id === s32(2)) { return "Editor focus"; }
    if (id === s32(3)) { return "Save"; }
    if (id === s32(4)) { return "Close"; }
    if (id === s32(5)) { return "Open Gnumeric"; }
    if (id === s32(6)) { return "Open CSV"; }
    return "Reopen";
}

export function Frame(frame: App.Frame): void {
    const session: Session = {slot: frame.slot, epoch: frame.epoch};
    const viewport = frame.viewport;
    State.Begin();
    <Box session={session} props={{key: u64(900000),
        bounds: {x: viewport.x, y: viewport.y, width: 200.0, height: viewport.height},
        fill: Grid.GridColor("surface"), border: Grid.GridColor("border-soft")}}/>;
    const x = viewport.x + float32(12);
    for (let id = s32(1); id <= s32(7); id += s32(1)) {
        if (Button({session: session, props: {key: u64(100) + u64(id), id: id,
            label: ActionLabel(id), bounds: {x: x, y: viewport.y + float32(16) + float32(id - s32(1)) * float32(36),
                width: 176.0, height: 28.0}}})) { State.Activate(id); }
    }
    State.UpdateLabels();
    for (let index = s32(0); index < s32(7); index += s32(1)) {
        <Text session={session} props={{key: u64(910001) + u64(index), text: State.Label(index), size: NativeText.Text14,
            bounds: {x: x, y: viewport.y + float32(280) + float32(index) * float32(24), width: 176.0, height: 20.0}}}/>;
    }
    const bounds: Rectangle = {x: viewport.x + float32(200), y: viewport.y,
        width: viewport.width - float32(200), height: viewport.height};
    const input = State.SampleInput(session, bounds);
    Editor.EditorFrame(session, bounds, State.EditorHandle(), input);
    State.AfterFrame();
}

export function main(): s32 {
    return App.Run({title: "workbook-host"}, Frame);
}
