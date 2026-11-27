import * as Model from "editor";
import * as Grid from "grid";
import * as NativeButton from "kryon/button";
import * as NativeMenu from "kryon/context_menu";
import * as NativeTabs from "kryon/tab_bar";
import * as NativeText from "kryon/text";
import { ButtonEmphasis, ButtonTone, ControlSize } from "kryon/control_props";
import { Color } from "kryon/drawing_props";
import { Rectangle } from "kryon/geometry";
import { KeyEscape } from "kryon/input_protocol";
import { Session } from "kryon/session";
import { TextUntilNul } from "std/text_buffer";
import { Box } from "kryon/tsx/box";
import { Button } from "kryon/tsx/button";
import { ContextMenu } from "kryon/tsx/context_menu";
import { ControlledTextField } from "kryon/tsx/text_field";
import { TabBar } from "kryon/tsx/tab_bar";
import { Text } from "kryon/tsx/text";
import { Image } from "kryon/tsx/image";
import * as NativeImage from "kryon/image";

import { Shape as Save } from "mingcute/mingcute_regular_save_2";
import { Shape as AddRow } from "mingcute/mingcute_extra_table_row_add";
import { Shape as RemoveRow } from "mingcute/mingcute_extra_table_row_remove";
import { Shape as AddColumn } from "mingcute/mingcute_extra_table_column_add";
import { Shape as RemoveColumn } from "mingcute/mingcute_extra_table_column_remove";
import { Shape as Left } from "mingcute/mingcute_regular_arrow_left";
import { Shape as Right } from "mingcute/mingcute_regular_arrow_right";
import { Shape as Up } from "mingcute/mingcute_regular_arrow_up";
import { Shape as Down } from "mingcute/mingcute_regular_arrow_down";

interface EditorRegion {
    session: Session;
    bounds: Rectangle;
    editor: NativeRef<Model.Editor>;
}

export interface EditorInputRegion {
    session: Session;
    bounds: Rectangle;
    editor: NativeRef<Model.Editor>;
    input: Model.EditorInput;
}

function IconButton(session: Session, base: s32, id: s32, x: float32, y: float32,
    icon: string, accent: boolean): boolean {
    const button: NativeButton.Props = {
        id: base + id, icon_shape: icon, icon_only: true,
        size: ControlSize.ControlSizeSmall, emphasis: ButtonEmphasis.ButtonEmphasisFilled,
        bounds: {x: x, y: y, width: 28.0, height: 28.0},
    };
    if (accent) { button.tone = ButtonTone.ButtonToneAccent; }
    return Button({session: session, props: button});
}

function Toolbar(view: EditorRegion): void {
    const session = view.session;
    const editor = view.editor;
    const x = view.bounds.x + float32(4);
    const y = view.bounds.y + float32(2);
    const base = Model.EditorControlKeyBase(editor);
    if (IconButton(session, base, s32(1001), x, y, Save, editor.dirty)) { Model.EditorSave(editor, true); }
    if (IconButton(session, base, s32(1002), x + float32(30), y, AddRow, false)) { Model.EditorInsertRow(editor); }
    if (IconButton(session, base, s32(1003), x + float32(60), y, RemoveRow, false)) { Model.EditorDeleteRow(editor); }
    if (IconButton(session, base, s32(1004), x + float32(90), y, AddColumn, false)) { Model.EditorInsertColumn(editor); }
    if (IconButton(session, base, s32(1005), x + float32(120), y, RemoveColumn, false)) { Model.EditorDeleteColumn(editor); }
    const chart: NativeButton.Props = {
        id: base + s32(1010), label: "Chart", size: ControlSize.ControlSizeSmall,
        bounds: {x: x + float32(150), y: y, width: 58.0, height: 28.0},
    };
    const chartPressed = Button({session: session, props: chart});
    if (chartPressed) { Model.EditorCreateFormulaChart(editor); }
    if (view.bounds.width < float32(600)) {
        const left = IconButton(session, base, s32(1006), x + float32(210), y, Left, false);
        const up = IconButton(session, base, s32(1007), x + float32(240), y, Up, false);
        const down = IconButton(session, base, s32(1008), x + float32(270), y, Down, false);
        const right = IconButton(session, base, s32(1009), x + float32(300), y, Right, false);
        Model.EditorMoveSelection(editor, left, up, down, right);
    }
}

function FormulaBar(session: Session, bounds: Rectangle, editor: NativeRef<Model.Editor>, state: NativeRef<Model.EditorFrameState>): void {
    const label: NativeText.Props = {
        key: editor.key_base + u64(3), text: "fx", size: NativeText.Text16, disabled: true,
        bounds: {x: bounds.x + float32(8), y: bounds.y + float32(35), width: 24.0, height: 20.0},
    };
    <Text session={session} props={label}/>;
    let result: Model.FormulaFieldResult = {};
    {
        const field = Model.EditorFormulaField(editor, bounds, state);
        field.value = TextUntilNul(nativeSlice(editor.edit));
        field.composition.text = TextUntilNul(nativeSlice(editor.edit_composition_text));
        if (field.focused) { field.input.text = TextUntilNul(nativeSlice(state.typed)); }
        const drawn = ControlledTextField({session: session, props: field});
        result = Model.EditorCopyFormulaFieldResult(drawn);
    }
    Model.EditorApplyFormulaField(editor, state.control, result);
}

function Outline(session: Session, key: u64, bounds: Rectangle, color: Color, thickness: s32): void {
    if (bounds.width <= float32(0) || bounds.height <= float32(0)) { return; }
    for (let line = s32(0); line < thickness; line += s32(1)) {
        const inset = float32(line);
        <Box session={session} props={{
            key: key + u64(line), border: color,
            bounds: {x: bounds.x + inset, y: bounds.y + inset,
                width: bounds.width - inset * float32(2), height: bounds.height - inset * float32(2)},
        }}/>;
    }
}

function GridSelection(session: Session, editor: NativeRef<Model.Editor>, table: Rectangle, grid: Model.EditorGrid): void {
    const accent = Grid.GridAccentColor();
    if (grid.here && editor.selection_axis === s32(0) && !editor.edit_focused) {
        Outline(session, editor.key_base + u64(400000), Model.EditorSelectionOutline(editor, table, grid), accent, s32(2));
        const handle = Model.FillHandleBounds(editor, table);
        if (handle.width > float32(0)) {
            <Box session={session} props={{
                key: editor.key_base + u64(400008), fill: accent, border: Grid.GridCellBackground(),
                bounds: {x: handle.x + float32(2), y: handle.y + float32(2), width: 7.0, height: 7.0},
            }}/>;
        }
    }
    Outline(session, editor.key_base + u64(400010), Model.EditorFillOutline(editor, table, grid), accent, s32(1));
    const ref: Model.EditorReferenceOutline = {};
    for (let index = s32(0); index < s32(32); index += s32(1)) {
        if (!Model.EditorReadReferenceOutline(editor, table, index, nativeRef(ref))) { break; }
        Outline(session, editor.key_base + u64(400100) + u64(index) * u64(4), ref.bounds, ref.color, s32(2));
    }
}

function InlineEditor(session: Session, editor: NativeRef<Model.Editor>, table: Rectangle): void {
    const text = TextUntilNul(nativeSlice(editor.edit));
    const bounds = Model.EditorInlineBounds(editor, table, text);
    if (bounds.width <= float32(0) || bounds.height <= float32(0)) { return; }
    const accent = Grid.GridAccentColor();
    <Box session={session} props={{key: editor.key_base + u64(400500), bounds: bounds,
        fill: Grid.GridCellBackground(), border: accent}}/>;
    <Text session={session} props={{
        key: editor.key_base + u64(400501), text: text, size: NativeText.Text14, wrap: NativeText.Wrap.TextWrapNone,
        bounds: {x: bounds.x + float32(4), y: bounds.y + float32(3),
            width: bounds.width - float32(8), height: bounds.height - float32(4)},
    }}/>;
    const caret = Model.EditorCaretX(editor, text);
    if (caret < bounds.width - float32(2)) {
        <Box session={session} props={{
            key: editor.key_base + u64(400502), fill: accent, border: accent,
            bounds: {x: bounds.x + caret, y: bounds.y + float32(5), width: 1.0, height: bounds.height - float32(10)},
        }}/>;
    }
}

function CellKey(editor: NativeRef<Model.Editor>, index: s32): u64 {
    return editor.key_base + u64(1000000) + u64(index) * u64(2);
}

function GridHeading(session: Session, editor: NativeRef<Model.Editor>, index: s32, bounds: Rectangle, selected: boolean, text: string): void {
    const fill = selected ? Grid.GridSelectedBackground() : Grid.GridHeaderBackground();
    Grid.GridSquare(session, CellKey(editor, index), bounds, editor.grid_right, editor.grid_bottom, fill, text, false);
}

function CellGrid(view: EditorInputRegion): void {
    const session = view.session;
    const editor = view.editor;
    const bounds = view.bounds;
    const table: Rectangle = {x: bounds.x, y: bounds.y + float32(60), width: bounds.width, height: bounds.height - float32(88)};
    const grid = Model.EditorPrepareGrid(editor, table, view.input);
    const headerWidth = float32(Grid.GridRowHeaderWidth);
    const headerHeight = float32(Grid.GridHeaderHeight);
    GridHeading(session, editor, s32(10), {x: table.x, y: table.y, width: headerWidth, height: headerHeight}, false, "");
    const columns = Model.Min(grid.columns, editor.scroll_column + editor.visible_columns);
    const rows = Model.Min(grid.rows, editor.scroll_row + editor.visible_rows);
    for (let column = editor.scroll_column; column < columns; column += s32(1)) {
        const width = Model.GridColWidth(editor, column);
        if (width <= float32(0)) { continue; }
        const name: Model.EditorColumnLabelBuffer = {};
        const text = Model.EditorColumnLabel(column, nativeSlice(name.bytes));
        const header: Rectangle = {x: Model.ColumnX(editor, table, column), y: table.y, width: width, height: headerHeight};
        const selected = grid.here && editor.selection_axis === s32(2) && column === editor.selected_column;
        GridHeading(session, editor, s32(20) + column - editor.scroll_column, header, selected, text);
    }
    for (let row = editor.scroll_row; row < rows; row += s32(1)) {
        const shown = row - editor.scroll_row;
        const y = Model.RowY(editor, table, row);
        const height = Model.GridRowHeight(editor, row);
        if (Model.RowHidden(editor, row)) {
            if (height > float32(0)) {
                GridHeading(session, editor, s32(1000) + shown, {x: table.x, y: y, width: table.width, height: height}, false, "");
                <Box session={session} props={{
                    key: editor.key_base + u64(500000) + u64(shown), fill: Grid.GridLineColor(),
                    bounds: {x: table.x, y: y + float32(2), width: table.width, height: 1.0},
                }}/>;
            }
            continue;
        }
        const digits: Model.EditorRowLabelBuffer = {};
        const label = Model.EditorRowLabel(row, nativeSlice(digits.bytes));
        const selected = grid.here && editor.selection_axis === s32(1) && row === editor.selected_row;
        GridHeading(session, editor, s32(1000) + shown, {x: table.x, y: y, width: headerWidth, height: height}, selected, label);
        for (let column = editor.scroll_column; column < columns; column += s32(1)) {
            const width = Model.GridColWidth(editor, column);
            if (width <= float32(0)) { continue; }
            const cell: Model.EditorGridCell = {};
            if (!Model.EditorReadGridCell(editor, table, grid, row, column, y, height, width, nativeRef(cell))) { continue; }
            const key = CellKey(editor, s32(10000) + shown * editor.visible_columns + column - editor.scroll_column);
            Grid.GridSquareStyled(session, key, cell.bounds, editor.grid_right, editor.grid_bottom,
                cell.fill, cell.line, TextUntilNul(nativeSlice(cell.display)), cell.has_fg, cell.fg, cell.bold, cell.italic,
                cell.align, cell.font_size, nativeRef(cell.style));
        }
    }
    GridSelection(session, editor, table, grid);
    Model.EditorDrawEmbeddedCharts(session, editor, table);
    InlineEditor(session, editor, table);
    Model.EditorHandleGridPointer(editor, table, grid, view.input);
}

function ColoredTabs(view: EditorInputRegion, shown: s32[], count: s32): boolean {
    const editor = view.editor;
    if (!Model.EditorHasColoredTabs(editor, shown, count)) { return false; }
    const bounds = Model.SheetTabsBounds(editor, view.bounds);
    let width = bounds.width / float32(count);
    if (width < float32(72)) { width = float32(72); }
    if (width > float32(104)) { width = float32(104); }
    let picked = s32(-1);
    for (let index = s32(0); index < count; index += s32(1)) {
        const sheet = nativeAt(shown, editor.tab_scroll + index);
        const style = Model.EditorSheetTabStyle(editor, sheet);
        const rect: Rectangle = {x: bounds.x + float32(index) * width, y: bounds.y, width: width, height: 28.0};
        if (!view.input.blocked && view.input.pointer.pressed &&
            Grid.GridInside(view.input.pointer.x, view.input.pointer.y, rect)) { picked = sheet; }
        <Box session={view.session} props={{key: editor.key_base + u64(1100) + u64(index),
            bounds: rect, fill: style.fill, border: {r: u8(0), g: u8(0), b: u8(0), a: u8(50)}}}/>;
        if (style.active) {
            <Box session={view.session} props={{key: editor.key_base + u64(410000) + u64(index),
                bounds: {x: rect.x + float32(1), y: rect.y, width: rect.width - float32(2), height: 3.0},
                fill: style.accent}}/>;
        }
        Model.EditorPaintSheetTabLabel(view.session, editor, sheet, rect, style);
    }
    Model.EditorPickSheet(editor, picked);
    return true;
}

// Native chart rendering supplies the cached texture and clipped source.
export function DrawEmbeddedChart(session: Session, image: NativeImage.Props, border: Color): void {
    <Image session={session} props={image}/>;
    <Box session={session} props={{key: image.key + u64(1), bounds: image.bounds, border: border}}/>;
}

function SheetTabs(view: EditorInputRegion): void {
    const session = view.session;
    const editor = view.editor;
    const bounds = view.bounds;
    const visible: Model.EditorTabList = {};
    const shown = Model.EditorShownSheets(editor, nativeSlice(visible.sheets));
    if (shown < s32(1)) { return; }
    const y = bounds.y + bounds.height - float32(28);
    <Box session={session} props={{
        key: editor.key_base + u64(1098), fill: Grid.GridHeaderBackground(), border: Grid.GridLineColor(),
        bounds: {x: bounds.x, y: y, width: bounds.width, height: 28.0},
    }}/>;
    const base = Model.EditorControlKeyBase(editor);
    const maxScroll = shown - s32(16);
    if (maxScroll > s32(0)) {
        const previous: NativeButton.Props = {
            id: base + s32(1096), label: "<", size: ControlSize.ControlSizeSmall, disabled: editor.tab_scroll <= s32(0),
            bounds: {x: bounds.x + float32(2), y: y + float32(2), width: 22.0, height: 24.0},
        };
        const previousPressed = Button({session: session, props: previous});
        if (previousPressed && editor.tab_scroll > s32(0)) { editor.tab_scroll = editor.tab_scroll - s32(1); }
        const next: NativeButton.Props = {
            id: base + s32(1097), label: ">", size: ControlSize.ControlSizeSmall, disabled: editor.tab_scroll >= maxScroll,
            bounds: {x: bounds.x + float32(26), y: y + float32(2), width: 22.0, height: 24.0},
        };
        const nextPressed = Button({session: session, props: next});
        if (nextPressed && editor.tab_scroll < maxScroll) { editor.tab_scroll = editor.tab_scroll + s32(1); }
    }
    const count = Model.Min(shown - editor.tab_scroll, s32(16));
    if (ColoredTabs(view, nativeSlice(visible.sheets), count)) { return; }
    let selected = s32(0);
    for (let index = s32(0); index < count; index += s32(1)) {
        const sheet = nativeAt(visible.sheets, editor.tab_scroll + index);
        nativeSet(visible.tabs, index, {key: editor.key_base + u64(1100) + u64(index),
            label: Model.EditorSheetTabName(editor, sheet)});
        if (sheet === editor.sheet_index) { selected = index; }
    }
    const bar: NativeTabs.Props = {
        key: editor.key_base + u64(1099), selected_index: selected,
        placement: NativeTabs.TabBarPlacement.TabBarBottom, bounds: Model.SheetTabsBounds(editor, bounds),
        min_tab_width: s32(72), max_tab_width: s32(104),
    };
    const result = TabBar({session: session, props: bar, tabs: nativeSlice(visible.tabs, s32(0), count)});
    if (result.selected_index >= s32(0) && result.selected_index < count) {
        Model.EditorPickSheet(editor, nativeAt(visible.sheets, editor.tab_scroll + result.selected_index));
    }
}

function GridMenu(session: Session, editor: NativeRef<Model.Editor>, escape: boolean): void {
    if (!editor.grid_menu_open) { return; }
    const items: Model.EditorGridMenuItems = {};
    const count = Model.EditorGridCommands(editor, nativeSlice(items.items));
    const props: NativeMenu.Props = {key: editor.key_base + u64(1200), open: true,
        x: editor.grid_menu_x, y: editor.grid_menu_y, escape: escape};
    const result = ContextMenu({session: session, props: props, items: nativeSlice(items.items, s32(0), count)});
    Model.EditorApplyGridCommand(editor, result);
}

function EditorStatus(view: EditorRegion): void {
    const bounds = view.bounds;
    const text = TextUntilNul(nativeSlice(view.editor.status));
    if (text.count === s64(0)) { return; }
    <Text session={view.session} props={{key: view.editor.key_base + u64(5), text: text, size: NativeText.Text14,
        bounds: {x: bounds.x + float32(180), y: bounds.y + bounds.height - float32(24),
            width: bounds.width - float32(184), height: 20.0},
    }}/>;
}

export function WorkbookEditor(view: EditorInputRegion): boolean {
    const state: Model.EditorFrameState = {};
    const editor = view.editor;
    const session = view.session;
    const bounds = view.bounds;
    if (!Model.EditorPrepareFrame(bounds, editor, view.input, nativeRef(state))) { return false; }
    <Box session={session} props={{key: editor.key_base + u64(7), bounds: bounds, fill: Grid.GridColor("canvas")}}/>;
    <Toolbar session={session} editor={editor} bounds={bounds}/>;
    state.was_editing = editor.edit_focused;
    FormulaBar(session, bounds, editor, nativeRef(state));
    Model.EditorApplyGridKeys(editor, nativeRef(state));
    <CellGrid session={session} editor={editor} bounds={bounds} input={state.input}/>;
    <SheetTabs session={session} editor={editor} bounds={bounds} input={state.input}/>;
    GridMenu(session, editor, state.key === KeyEscape);
    Model.EditorAutosave(editor);
    <EditorStatus session={session} editor={editor} bounds={bounds}/>;
    Model.EditorFinishFrame(editor, state.input.pointer);
    return state.handled;
}
