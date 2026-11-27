// Typed JSX bindings for the public draw API in Workbook's Kryon pin.
import * as NativeBox from "kryon/box";
import * as NativeButton from "kryon/button";
import * as NativeContextMenu from "kryon/context_menu";
import * as NativeTabBar from "kryon/tab_bar";
import * as NativeText from "kryon/text";
import * as NativeTextField from "kryon/text_field";
import { Session } from "kryon/session";

export interface BoxProps {
    session: Session;
    props: NativeBox.Props;
}

export function Box(widget: BoxProps): void {
    NativeBox.Draw(widget.session, widget.props);
}

export interface ButtonProps {
    session: Session;
    props: NativeButton.Props;
}

export function Button(widget: ButtonProps): s32 {
    return NativeButton.Draw(widget.session, widget.props);
}

export interface TextProps {
    session: Session;
    props: NativeText.Props;
}

export function Text(widget: TextProps): void {
    NativeText.Draw(widget.session, widget.props);
}

export interface TextFieldProps {
    session: Session;
    props: NativeTextField.Props;
}

export function ControlledTextField(widget: TextFieldProps): NativeTextField.Result {
    return NativeTextField.Draw(widget.session, widget.props);
}

export interface Tabs {
    session: Session;
    props: NativeTabBar.Props;
    tabs: NativeTabBar.Tab[];
}

export function TabBar(widget: Tabs): NativeTabBar.Result {
    return NativeTabBar.Draw(widget.session, widget.props, widget.tabs);
}

export interface ContextItems {
    session: Session;
    props: NativeContextMenu.Props;
    items: NativeContextMenu.ContextMenuItem[];
}

export function ContextMenu(widget: ContextItems): NativeContextMenu.ContextMenuResult {
    return NativeContextMenu.Draw(widget.session, widget.props, widget.items);
}
