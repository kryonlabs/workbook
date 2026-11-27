import * as Grid from "grid";
import * as Model from "editor";
import * as Menu from "kryon/menu";
import * as Popup from "kryon/context_menu";
import * as NativeText from "kryon/text";
import { ControlSize } from "kryon/control_props";
import { Rectangle } from "kryon/geometry";
import { Session } from "kryon/session";
import { WorkbookEditor } from "editor_view";
import { Box } from "kryon/tsx/box";
import { Button } from "kryon/tsx/button";
import { ContextMenu } from "kryon/tsx/context_menu";
import { Text } from "kryon/tsx/text";

export function DrawDocument(session: Session, menuBounds: Rectangle, editorBounds: Rectangle,
    editor: NativeRef<Model.Editor>, input: Model.EditorInput): void {
    <Box session={session} props={{key: u64(990000), bounds: menuBounds, fill: Grid.GridColor("canvas")}}/>;
    <WorkbookEditor session={session} bounds={editorBounds} editor={editor} input={input}/>;
}

export function DrawUpdateMessage(session: Session, bounds: Rectangle): void {
    <Text session={session} props={{bounds: bounds, text: "Update Werkzeug to open Workbook.", size: NativeText.Text16}}/>;
}

export function DrawHostedMenu(session: Session, props: Menu.Props, groups: Menu.MenuGroup[],
    state: NativeRef<Menu.MenuState>, items: Popup.ContextMenuItem[]): s32 {
    if (!state.initialized) { Menu.MenuStateInit(state); }
    const bounds = props.bounds;
    for (let index = s32(0); index < s32(groups.count); index += s32(1)) {
        const pressed = Button({session: session, props: {
            key: u64(1310) + u64(index), label: nativeAt(groups, index).label, size: ControlSize.ControlSizeSmall,
            bounds: {x: bounds.x + float32(2) + float32(index) * float32(58), y: bounds.y, width: 58.0, height: 26.0},
        }});
        if (pressed) { state.open_index = state.open_index === index ? s32(-1) : index; }
    }
    const open = state.open_index;
    if (open < s32(0) || open >= s32(groups.count)) { return s32(0); }
    const commands = nativeAt(groups, open).items;
    const count = s32(commands.count);
    for (let index = s32(0); index < count; index += s32(1)) {
        const item = nativeAt(commands, index);
        nativeSet(items, index, {id: item.id, label: item.label, disabled: item.disabled,
            separator: item.kind === Menu.MenuItemKind.MenuSeparator});
    }
    const result = ContextMenu({session: session, items: nativeSlice(items, s32(0), count), props: {
        key: u64(1350) + u64(open), open: true, escape: props.input.escape,
        x: bounds.x + float32(2) + float32(open) * float32(58), y: bounds.y + float32(26),
    }});
    if (!result.open) { state.open_index = s32(-1); }
    return result.activated_id;
}
