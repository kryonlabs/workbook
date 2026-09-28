#!/usr/bin/env python3
"""Exercise the Workbook desktop app on a private X display.

The app runs only under xvfb-run with DISPLAY and WAYLAND_DISPLAY removed, so
the test can never send input to a real desktop. It checks one captured frame,
then edits cells with the keyboard and mouse, saves with Ctrl+S, and reads the
saved file back with the headless `cell` driver.

    python3 tests/desktop_test.py           # run the whole test
"""

import csv
import os
import shutil
import signal
import struct
import subprocess
import sys
import tempfile
import time
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
APP = ROOT / "build" / "workbook-desktop"
CELL = ROOT / "build" / "cell"
FIXTURE = ROOT / "tests" / "fixtures" / "operators.gnumeric"
WINDOW_NAME = "Kryon Ziran"

# Grid geometry from src/app.zi, in window pixels.
GRID_LEFT = 24 + 52
GRID_TOP = 98 + 30
COLUMN_WIDTH = 110
ROW_HEIGHT = 28

# Centers of the toolbar buttons (y is the same for all) and of the first
# sheet tabs, which sit under the grid.
TOOLBAR_Y = 30
TOOLBAR = {"Save": 282, "Insert row": 370, "Delete row": 474,
           "Insert column": 588, "Delete column": 712}
TAB_Y = 600 - 78 + 16
TAB_WIDTH = 104
TAB_LEFT = 24

BACKGROUND = (248, 250, 252)
HEADER = (0xED, 0xF4, 0xFC)
SELECTED = (0xDC, 0xEB, 0xFF)
WHITE = (255, 255, 255)


def private_environment():
    env = os.environ.copy()
    for name in ("DISPLAY", "WAYLAND_DISPLAY", "XAUTHORITY", "KRYON_CAPTURE_PATH"):
        env.pop(name, None)
    return env


def cell_origin(row, column):
    return GRID_LEFT + column * COLUMN_WIDTH, GRID_TOP + row * ROW_HEIGHT


def cell_center(row, column):
    x, y = cell_origin(row, column)
    return x + COLUMN_WIDTH // 2, y + ROW_HEIGHT // 2


# ------------------------------------------------------------------ PNG


def read_png(path):
    """Return (width, height, pixel) for an 8-bit RGB or RGBA PNG."""
    data = Path(path).read_bytes()
    assert data[:8] == b"\x89PNG\r\n\x1a\n", "not a PNG"
    position = 8
    header = None
    compressed = b""
    while position < len(data):
        length, kind = struct.unpack(">I4s", data[position:position + 8])
        body = data[position + 8:position + 8 + length]
        position += 12 + length
        if kind == b"IHDR":
            header = struct.unpack(">IIBBBBB", body)
        elif kind == b"IDAT":
            compressed += body
    width, height, depth, color, _, _, interlace = header
    assert depth == 8 and color in (2, 6) and interlace == 0, header
    channels = 3 if color == 2 else 4
    stride = width * channels
    raw = zlib.decompress(compressed)
    rows = []
    previous = bytearray(stride)
    at = 0
    for _ in range(height):
        kind = raw[at]
        line = bytearray(raw[at + 1:at + 1 + stride])
        at += 1 + stride
        for i in range(stride):
            left = line[i - channels] if i >= channels else 0
            up = previous[i]
            corner = previous[i - channels] if i >= channels else 0
            if kind == 1:
                line[i] = (line[i] + left) & 255
            elif kind == 2:
                line[i] = (line[i] + up) & 255
            elif kind == 3:
                line[i] = (line[i] + (left + up) // 2) & 255
            elif kind == 4:
                estimate = left + up - corner
                distances = (abs(estimate - left), abs(estimate - up),
                             abs(estimate - corner))
                nearest = (left, up, corner)[distances.index(min(distances))]
                line[i] = (line[i] + nearest) & 255
        rows.append(line)
        previous = line

    def pixel(x, y):
        offset = x * channels
        return tuple(rows[y][offset:offset + 3])

    return width, height, pixel


def dark_pixels(pixel, left, top, width, height):
    count = 0
    for y in range(top, top + height):
        for x in range(left, left + width):
            red, green, blue = pixel(x, y)
            if red + green + blue < 3 * 110:
                count += 1
    return count


# ---------------------------------------------------------------- checks


def run(command, env, **options):
    return subprocess.run(command, env=env, check=True, capture_output=True,
                          text=True, **options)


def check_frame(work, env):
    data = work / "frame"
    data.mkdir()
    shutil.copy(FIXTURE, data / "workbook.gnumeric")
    capture = work / "frame.png"
    frame_env = dict(env, WORKBOOK_DIR=str(data), KRYON_CAPTURE_PATH=str(capture))
    subprocess.run(["xvfb-run", "-a", str(APP)], env=frame_env, check=True,
                   capture_output=True, text=True, timeout=120)
    width, height, pixel = read_png(capture)
    assert (width, height) == (960, 600), (width, height)
    assert pixel(10, 10) == BACKGROUND, pixel(10, 10)
    x, y = cell_origin(0, 0)
    assert pixel(x + 90, y + 20) == SELECTED, ("selected A1", pixel(x + 90, y + 20))
    x, y = cell_origin(0, 1)
    assert pixel(x + 90, y + 20) == WHITE, ("unselected B1", pixel(x + 90, y + 20))
    assert pixel(GRID_LEFT + 90, 98 + 6) == HEADER, ("column header",)
    # The fixture fills A1 ("10") and B1 ("alpha"); column F is empty.
    x, y = cell_origin(0, 0)
    assert dark_pixels(pixel, x + 2, y + 2, COLUMN_WIDTH - 4, ROW_HEIGHT - 4) > 10, "A1 text"
    x, y = cell_origin(0, 5)
    assert dark_pixels(pixel, x + 2, y + 2, COLUMN_WIDTH - 4, ROW_HEIGHT - 4) == 0, "F1 empty"
    print("frame: layout, selection, and cell text rendered")


PRIVATE_MARKER = "WORKBOOK_TEST_PRIVATE_DISPLAY"


class Session:
    """The running app and the xdotool calls that drive it.

    It runs only inside the private display that xvfb-run created for this
    test: the marker is set by the test itself when it starts xvfb-run, and
    the driver refuses to start without it.
    """

    def __init__(self, data):
        assert os.environ.get(PRIVATE_MARKER) == "1", (
            "the driver sends input events; run tests/desktop_test.py, which "
            "starts it on a private display")
        self.data = data
        self.env = os.environ.copy()
        self.env.pop("KRYON_CAPTURE_PATH", None)
        self.env["WORKBOOK_DIR"] = str(data)
        self.app = subprocess.Popen([str(APP)], env=self.env,
                                    stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, text=True)
        self.window = self.find_window()
        subprocess.run(["xdotool", "windowfocus", self.window], check=True)
        time.sleep(0.5)

    def find_window(self):
        for _ in range(100):
            found = subprocess.run(["xdotool", "search", "--name", WINDOW_NAME],
                                   capture_output=True, text=True)
            if found.returncode == 0 and found.stdout.strip():
                return found.stdout.splitlines()[0]
            if self.app.poll() is not None:
                raise AssertionError("the app exited early: " + self.app.stderr.read())
            time.sleep(0.1)
        raise AssertionError("the app window did not appear")

    def key(self, *names):
        for name in names:
            subprocess.run(["xdotool", "key", name], check=True)
            time.sleep(0.2)

    def type(self, text):
        subprocess.run(["xdotool", "type", "--delay", "90", text], check=True)
        time.sleep(0.25)

    def click(self, row, column, repeat=1):
        x, y = cell_center(row, column)
        if repeat == 1:
            self.press(x, y)
            return
        subprocess.run(["xdotool", "mousemove", "--window", self.window,
                        str(x), str(y)], check=True)
        time.sleep(0.2)
        subprocess.run(["xdotool", "click", "--repeat", str(repeat), "--delay",
                        "90", "1"], check=True)
        time.sleep(0.3)

    def press(self, x, y):
        """Click a button. It is held for a few frames, as a hand holds it:
        a Kryon button activates on a release that follows a press it saw."""
        subprocess.run(["xdotool", "mousemove", "--window", self.window,
                        str(x), str(y)], check=True)
        time.sleep(0.2)
        subprocess.run(["xdotool", "mousedown", "1"], check=True)
        time.sleep(0.12)
        subprocess.run(["xdotool", "mouseup", "1"], check=True)
        time.sleep(0.3)

    def toolbar(self, name):
        self.press(TOOLBAR[name], TOOLBAR_Y)

    def tab(self, index):
        self.press(TAB_LEFT + TAB_WIDTH * index + TAB_WIDTH // 2, TAB_Y)

    def save(self):
        self.key("ctrl+s")
        time.sleep(0.4)

    def sizes(self):
        """Each sheet's (rows, columns) in the saved workbook, by name."""
        saved = self.data / "workbook.gnumeric"
        output = run([str(CELL), "info", str(saved)], self.env).stdout
        sizes = {}
        for line in output.splitlines()[1:]:
            name, _, rest = line.rpartition(": ")
            rows, _, tail = rest.partition(" rows x ")
            sizes[name] = (int(rows), int(tail.split()[0]))
        return sizes

    def rows(self):
        """The saved workbook, evaluated: one list of cell texts per row,
        without the empty cells that pad a row to the sheet's width."""
        saved = self.data / "workbook.gnumeric"
        assert saved.exists(), "Ctrl+S did not write workbook.gnumeric"
        output = run([str(CELL), "eval", str(saved)], self.env).stdout
        rows = []
        for cells in csv.reader(output.splitlines()):
            while cells and cells[-1] == "":
                cells.pop()
            rows.append(cells)
        return rows

    def close(self):
        if self.app.poll() is None:
            self.app.send_signal(signal.SIGTERM)
        try:
            self.app.communicate(timeout=10)
        except subprocess.TimeoutExpired:
            self.app.kill()
            self.app.communicate()
        return self.app.returncode


def expect(session, expected, message):
    rows = session.rows()
    assert rows[:len(expected)] == expected, (message, rows)
    print("edit:", message)


def drive(data):
    session = Session(Path(data))
    try:
        # Typing over the selected cell starts an edit; Enter commits and
        # moves down. A formula recalculates from the cell above it.
        session.type("5")
        session.key("Return")
        session.type("=A1*2")
        session.key("Return")
        session.save()
        expect(session, [["5"], ["10"]], "typed values and a formula")

        # A click selects a cell; typing replaces its value.
        session.click(2, 2)
        session.type("hello")
        session.key("Return")
        session.save()
        expect(session, [["5"], ["10"], ["", "", "hello"]], "click, type, and save")

        # Escape leaves a cell as it was.
        session.click(0, 1)
        session.type("zzz")
        session.key("Escape")
        session.save()
        expect(session, [["5"], ["10"], ["", "", "hello"]], "Escape cancels an edit")

        # A double click edits the existing formula in place.
        session.click(1, 0, repeat=2)
        session.type("+1")
        session.key("Return")
        session.save()
        expect(session, [["5"], ["11"], ["", "", "hello"]], "double click edits a formula")

        # Delete clears the selected cell; the formula below follows it. The
        # export starts at the first row that shows anything, which is now A2.
        session.click(0, 0)
        session.key("Delete")
        session.save()
        expect(session, [["1"], ["", "", "hello"]], "Delete clears a cell")

        # The toolbar buttons edit the sheet's structure at the selection.
        # "hello" sits in row 3 and column C, so the sheet is 3 by 3.
        assert session.sizes() == {"Sheet1": (3, 3)}, session.sizes()
        session.click(1, 1)
        session.toolbar("Insert row")
        session.toolbar("Save")
        assert session.sizes() == {"Sheet1": (4, 3)}, session.sizes()
        session.toolbar("Delete row")
        session.toolbar("Save")
        assert session.sizes() == {"Sheet1": (3, 3)}, session.sizes()
        session.toolbar("Insert column")
        session.toolbar("Save")
        assert session.sizes() == {"Sheet1": (3, 4)}, session.sizes()
        session.toolbar("Delete column")
        session.toolbar("Save")
        assert session.sizes() == {"Sheet1": (3, 3)}, session.sizes()
        print("edit: toolbar buttons insert and delete rows and columns")
    finally:
        status = session.close()
    assert status == 0, ("the app did not exit cleanly", status)
    print("session: the app exited cleanly on SIGTERM")


TWO_SHEETS = """<?xml version="1.0" encoding="UTF-8"?>
<gnm:Workbook xmlns:gnm="http://www.gnumeric.org/v10.dtd">
  <gnm:SheetNameIndex><gnm:SheetName>First</gnm:SheetName><gnm:SheetName>Second</gnm:SheetName></gnm:SheetNameIndex>
  <gnm:Sheets>
    <gnm:Sheet><gnm:Name>First</gnm:Name><gnm:MaxCol>0</gnm:MaxCol><gnm:MaxRow>0</gnm:MaxRow>
      <gnm:Cells><gnm:Cell Row="0" Col="0" ValueType="40">1</gnm:Cell></gnm:Cells>
    </gnm:Sheet>
    <gnm:Sheet><gnm:Name>Second</gnm:Name><gnm:MaxCol>0</gnm:MaxCol><gnm:MaxRow>0</gnm:MaxRow>
      <gnm:Cells><gnm:Cell Row="0" Col="0" ValueType="40">2</gnm:Cell></gnm:Cells>
    </gnm:Sheet>
  </gnm:Sheets>
</gnm:Workbook>
"""


def drive_tabs(data):
    (Path(data) / "workbook.gnumeric").write_text(TWO_SHEETS)
    session = Session(Path(data))
    try:
        assert session.sizes() == {"First": (1, 1), "Second": (1, 1)}, session.sizes()
        # The second tab makes that sheet current: an edit lands in it.
        session.tab(1)
        session.click(2, 1)
        session.type("7")
        session.key("Return")
        session.save()
        assert session.sizes() == {"First": (1, 1), "Second": (3, 2)}, session.sizes()
        # And back: the first sheet takes the next edit.
        session.tab(0)
        session.click(0, 3)
        session.type("4")
        session.key("Return")
        session.save()
        assert session.sizes() == {"First": (1, 4), "Second": (3, 2)}, session.sizes()
        print("tabs: each tab edits its own sheet")
    finally:
        status = session.close()
    assert status == 0, ("the app did not exit cleanly", status)


def main():
    if len(sys.argv) == 3 and sys.argv[1] == "--drive":
        drive(sys.argv[2])
        return
    if len(sys.argv) == 3 and sys.argv[1] == "--drive-tabs":
        drive_tabs(sys.argv[2])
        return
    for tool in ("xvfb-run", "xdotool"):
        assert shutil.which(tool), f"install {tool} to test the desktop app"
    for path in (APP, CELL):
        assert path.exists(), f"build first: {path} is missing"
    env = private_environment()
    with tempfile.TemporaryDirectory(prefix="workbook-desktop-") as directory:
        work = Path(directory)
        check_frame(work, env)
        data = work / "edit"
        data.mkdir()
        subprocess.run(["xvfb-run", "-a", "env", PRIVATE_MARKER + "=1",
                        sys.executable, __file__, "--drive", str(data)],
                       env=env, check=True, timeout=240)
        tabs = work / "tabs"
        tabs.mkdir()
        subprocess.run(["xvfb-run", "-a", "env", PRIVATE_MARKER + "=1",
                        sys.executable, __file__, "--drive-tabs", str(tabs)],
                       env=env, check=True, timeout=240)
    print("desktop app: rendering, keyboard and mouse editing, and saving passed")


if __name__ == "__main__":
    main()
