import { Color } from "kryon/drawing_props";
import { Rectangle } from "kryon/geometry";
import { Session } from "kryon/session";
import * as NativeText from "kryon/text";
import { TextAlign } from "kryon/text_align";
import { Box } from "kryon/tsx/box";
import { Text } from "kryon/tsx/text";
import { TabBar } from "kryon/tsx/tab_bar";
import * as Grid from "grid";
import * as NativeTabs from "kryon/tab_bar";

export function DrawCellBackground(session: Session, key: u64, bounds: Rectangle, fill: Color, line: Color): void {
    <Box session={session} props={{key: key, bounds: bounds, fill: fill, border: line}}/>;
}

export function DrawCellLabel(session: Session, key: u64, bounds: Rectangle, text: string, size: s32, align: s32): void {
    const label: NativeText.Props = {
        key: key + u64(1), text: text, size: size, wrap: NativeText.Wrap.TextWrapNone,
        bounds: {x: bounds.x + float32(4), y: bounds.y + float32(3),
            width: bounds.width - float32(8), height: bounds.height - float32(4)},
    };
    if (align === s32(1)) { label.align = TextAlign.TextAlignCenter; }
    else if (align === s32(2)) { label.align = TextAlign.TextAlignEnd; }
    <Text session={session} props={label}/>;
}

// The read-only embeddable grid uses the same cell primitives as the editor.
export function WorkbookGrid(session: Session, bounds: Rectangle, view: NativeRef<Grid.GridView>, frame: Grid.GridFrame): void {
    if (frame.sheets < s32(1)) { return; }
    const table = frame.table;
    const headerWidth = float32(Grid.GridRowHeaderWidth);
    const headerHeight = float32(Grid.GridHeaderHeight);
    const rowHeight = float32(Grid.GridRowHeight);
    Grid.GridSquare(session, view.key, {x: table.x, y: table.y, width: headerWidth, height: headerHeight},
        frame.right, frame.bottom, Grid.GridHeaderBackground(), "", false);
    for (let shown = s32(0); shown < view.visible_columns; shown += s32(1)) {
        const column = view.scroll_column + shown;
        Grid.GridSquare(session, view.key + u64(s32(2) + shown * s32(2)),
            {x: Grid.GridColumnX(view, table.x, column), y: table.y, width: Grid.GridWidth(view, column), height: headerHeight},
            frame.right, frame.bottom, Grid.GridHeaderBackground(), Grid.GridColumnName(column), false);
    }
    for (let shown = s32(0); shown < frame.shown_rows; shown += s32(1)) {
        const row = shown < frame.frozen ? shown : view.scroll_row + shown;
        if (view.trim && row >= frame.rows) { break; }
        const heading = shown < frame.frozen || (view.total_rows > s32(0) && row >= frame.rows - view.total_rows && row < frame.rows);
        const fill = heading ? Grid.GridHeaderBackground() : Grid.GridCellBackground();
        const y = table.y + headerHeight + float32(shown) * rowHeight;
        Grid.GridSquare(session, view.key + u64(s32(2000) + shown * s32(2)),
            {x: table.x, y: y, width: headerWidth, height: rowHeight}, frame.right, frame.bottom,
            Grid.GridHeaderBackground(), Grid.GridRowLabel(row), false);
        for (let index = s32(0); index < view.visible_columns; index += s32(1)) {
            const column = view.scroll_column + index;
            const display: Grid.GridTextBuffer = {};
            const label = Grid.GridReadCellLabel(frame, view, row, column, nativeSlice(display.bytes));
            Grid.GridSquare(session, view.key + u64(s32(100000) + (shown * s32(Grid.GridColumnsMax) + index) * s32(2)),
                {x: Grid.GridColumnX(view, table.x, column), y: y, width: Grid.GridWidth(view, column), height: rowHeight},
                frame.right, frame.bottom, fill, label.text, label.numeric);
        }
    }
    if (frame.sheets < s32(2)) { return; }
    const list: Grid.GridTabList = {};
    Grid.GridReadTabs(frame, view, nativeRef(list));
    <Box session={session} props={{key: view.key + u64(89998),
        bounds: {x: bounds.x, y: frame.bottom, width: bounds.width, height: float32(Grid.GridTabsHeight)},
        fill: Grid.GridHeaderBackground(), border: Grid.GridLineColor()}}/>;
    const width = float32(140) * float32(list.count);
    const props: NativeTabs.Props = {key: view.key + u64(89999),
        placement: NativeTabs.TabBarPlacement.TabBarBottom, selected_index: list.selected,
        bounds: {x: bounds.x, y: frame.bottom, width: width < bounds.width ? width : bounds.width, height: float32(Grid.GridTabsHeight)},
        min_tab_width: s32(96), max_tab_width: s32(160)};
    const result = TabBar({session: session, props: props, tabs: nativeSlice(list.tabs, s32(0), list.count)});
    if (result.selected_index >= s32(0) && result.selected_index < list.count) {
        const selected = nativeAt(list.sheets, result.selected_index);
        if (selected !== view.sheet) {
            view.sheet = selected;
            view.scroll_row = s32(0);
            view.scroll_column = s32(0);
        }
    }
}
