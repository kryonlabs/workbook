import { Color } from "kryon/drawing_props";
import { Rectangle } from "kryon/geometry";
import { Session } from "kryon/session";
import * as NativeText from "kryon/text";
import { TextAlign } from "kryon/text_align";
import { Box, Text } from "widgets";

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
