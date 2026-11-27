// The native desktop menu; hosted documents compose their own buttons/popup.
import * as Menu from "kryon/menu";
import { Menu as MenuBar } from "kryon/tsx/menu";
import { Session } from "kryon/session";

export function DrawMenu(session: Session, props: Menu.Props, groups: Menu.MenuGroup[],
    state: NativeRef<Menu.MenuState>): s32 {
    const result = MenuBar({session: session, props: props, groups: groups, state: state});
    return result.activated_id;
}
