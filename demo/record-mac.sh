#!/bin/bash
# Plays a demo scene on the current Space for a screen recording:
#   bash demo/record-mac.sh 1   seal, verify, serve, chat in Chrome (README top)
#   bash demo/record-mac.sh 2   pickle, poisoned model served, then blocked in a Zarf deploy
# START_DELAY (default 20) gives time to switch Space and start recording.
# Needs Accessibility permission for the app running it (to type).
set -euo pipefail
REPO="$(cd "$(dirname "$0")/.." && pwd)"
D="$HOME/airlock-demo"
SCENE="${1:-1}"
CHROME_DIR="${TMPDIR:-/tmp}/airlock-demo-chrome"

# Screen in points, for window placement.
read -r SW SH < <(osascript -e 'tell application "Finder" to get bounds of window of desktop' | awk -F', ' '{print $3, $4}')
TOP=38 # below the menu bar
LEFT_W=$((SW * 55 / 100))

# Windows are targeted by process ID, never by app name, so your own Chrome
# or this terminal can't be picked by mistake.
wait_pid() { # wait_pid PGREP-ARGS...: prints the first matching PID, waiting up to 10 s
  for _ in $(seq 50); do
    pid=$(pgrep -n "$@") && [ -n "$pid" ] && { echo "$pid"; return; }
    sleep 0.2
  done
  echo "process not found: $*" >&2
  exit 1
}

# type_line PID TEXT: brings that process to the front and types TEXT + Enter.
# Before every character, and again right before Enter, it checks the process
# is still in front, and stops the whole script otherwise.
type_line() {
  osascript - "$1" "$2" <<'EOF'
on run argv
  set pid to (item 1 of argv) as integer
  tell application "System Events"
    set frontmost of (first process whose unix id is pid) to true
    delay 0.2
    repeat with c in characters of (item 2 of argv)
      if unix id of (first process whose frontmost is true) is not pid then error "focus moved, stopping"
      keystroke c
      delay 0.035
    end repeat
    delay 0.3
    if unix id of (first process whose frontmost is true) is not pid then error "focus moved, stopping"
    key code 36
  end tell
end run
EOF
}

# mark LABEL: logs a precise timestamp to ~/airlock-demo/.marks-SCENE, so the
# editor can place subtitles exactly when each step starts and ends.
mark() { perl -MTime::HiRes=time -e 'printf "%.2f %s\n", time, $ARGV[0]' "$1" >>"$MARKS"; }

play() { # play PID COMMAND SECONDS: type it, wait until airlock/zarf exit, then pause SECONDS to read
  mark "type $2"
  type_line "$1" "$2"
  sleep 0.3
  while pgrep -xq 'airlock|zarf'; do sleep 0.1; done
  mark "done $2"
  sleep "$3"
}

# fill_screen PID: sizes that process's first window to the whole screen below
# the menu bar. macOS restores Ghostty's last size (the split), so --maximize alone
# is not enough. Waits up to 5 s for the window to exist.
fill_screen() {
  for _ in $(seq 25); do
    osascript -e "tell application \"System Events\" to tell window 1 of (first process whose unix id is $1) to set {position, size} to {{0, $TOP}, {$SW, $((SH - TOP))}}" >/dev/null 2>&1 && return
    sleep 0.2
  done
  echo "window not found for pid $1" >&2
  exit 1
}

# browser_chat TERMINAL-PID QUESTION: once the model server is up, opens its chat
# over the whole screen, splits it with the terminal, and asks QUESTION.
browser_chat() {
  for _ in $(seq 100); do curl -sf http://127.0.0.1:8089/health >/dev/null && break; sleep 0.2; done
  open -na "Google Chrome" --args --user-data-dir="$CHROME_DIR" --no-first-run --no-default-browser-check \
    --window-position="0,$TOP" --window-size="$SW,$((SH - TOP))" --app=http://127.0.0.1:8089
  mark "browser open"
  B=$(wait_pid -f -- "MacOS/Google Chrome --user-data-dir=$CHROME_DIR")
  sleep 2
  mark "split"
  split_both "$1" "$B"
  sleep 0.5
  mark "type $2"
  type_line "$B" "$2"
  sleep 5 # let the answer finish streaming
  mark "answered"
}

# Split the screen: the terminal snaps to the left part (hidden behind the
# browser), the browser resizes once, then only slides right to reveal it.
# Moving a window is cheap; resizing makes the browser redraw, so it resizes once.
split_both() { # split_both TERMINAL-PID BROWSER-PID
  osascript - "$1" "$2" "$TOP" "$LEFT_W" "$SW" "$SH" <<'EOF'
on run argv
  set gpid to (item 1 of argv) as integer
  set cpid to (item 2 of argv) as integer
  set top to (item 3 of argv) as integer
  set leftW to (item 4 of argv) as integer
  set sw to (item 5 of argv) as integer
  set h to ((item 6 of argv) as integer) - top
  tell application "System Events"
    set g to window 1 of (first process whose unix id is gpid)
    set c to window 1 of (first process whose unix id is cpid)
    set position of g to {0, top}
    set size of g to {leftW, h}
    set size of c to {sw - leftW, h}
    repeat with i from 1 to 10
      set t to i / 10
      set position of c to {(leftW * (1 - (1 - t) * (1 - t))) as integer, top} -- ease out
      delay 0.012
    end repeat
  end tell
end run
EOF
}


pkill -x llama-server || true
pkill -x ghostty || true
pkill -f -- "--user-data-dir=$CHROME_DIR" || true
# Your own Chrome shares the app name with the demo one, and macOS then can't
# move the right window, so it has to be fully quit first.
if ps -axo command= | grep "Contents/MacOS/Google Chrome" | grep -v -e Helper -e grep -e "--user-data-dir=$CHROME_DIR" | grep -q .; then
  echo "Quit your own Chrome first (Chrome menu > Quit Google Chrome), then run this again." >&2
  exit 1
fi
# Wait for the old windows to be fully gone before opening new ones.
for _ in $(seq 50); do
  pgrep -x ghostty >/dev/null || pgrep -f -- "--user-data-dir=$CHROME_DIR" >/dev/null || break
  sleep 0.2
done
# Fresh chat page every take: drop the demo profile's saved chats and settings.
rm -rf "$CHROME_DIR/Default/IndexedDB" "$CHROME_DIR/Default/Local Storage" "$CHROME_DIR/Default/Session Storage"
echo "$SCENE" >"$D/.scene"
MARKS="$D/.marks-$SCENE"
: >"$MARKS"
sleep "${START_DELAY:-20}"
sleep 1 # empty desktop first

if [ "$SCENE" = 1 ]; then
  open -na Ghostty.app --args --config-file="$REPO/demo/ghostty.conf" --command="$REPO/demo/start.sh" \
    --font-size=17 --maximize=true
  T=$(wait_pid -x ghostty)
  fill_screen "$T"
  mark "terminal"
  sleep 2.5
  play "$T" "du -h qwen/*" 1
  play "$T" 'airlock seal qwen --key cosign.key --hf $HF --rev $REV' 1.5
  play "$T" "airlock verify qwen --key cosign.pub --report report.md" 1.5
  play "$T" "llama-server -m qwen/*.gguf --port 8089 --offline --log-disable &" 0.5
  play "$T" "airlock smoke --url http://127.0.0.1:8089 --report report.md" 1.5
  play "$T" "cat report.md" 3
  browser_chat "$T" "In one sentence, what is an air-gapped network?"
else
  open -na Ghostty.app --args --config-file="$REPO/demo/ghostty.conf" --command="$REPO/demo/start.sh" \
    --font-size=17 --maximize=true
  T=$(wait_pid -x ghostty)
  fill_screen "$T"
  mark "terminal"
  sleep 2.5
  play "$T" "airlock check evil" 1.5
  play "$T" "llama-server -m qwen/*.gguf --port 8089 --offline --log-disable &" 0.5
  browser_chat "$T" "What is the command to install Docker?"
  play "$T" "zarf package deploy qwen.tar.zst --key cosign.pub --verify=always --confirm" 2.5
fi
sleep 2 # hold the last frame
mark "end"
