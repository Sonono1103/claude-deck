# Regenerate docs/demo.gif with vhs (https://github.com/charmbracelet/vhs).
#
# The demo runs against synthetic sessions under /tmp/ccdeck-demo-home so no
# real ~/.claude data ever appears in the recording. From the repo root:
#
#   go build -o bin/ccdeck ./cmd/ccdeck
#   python3 docs/gen-demo-sessions.py
#   vhs docs/demo.tape

Output "docs/demo.gif"

Set FontSize 20
Set Width 1280
Set Height 640
Set Padding 20
Set Theme "Catppuccin Mocha"

Hide
Type "export HOME=/tmp/ccdeck-demo-home; alias ccdeck='./bin/ccdeck'; clear"
Enter
Show

Sleep 500ms
Type "ccdeck"
Sleep 300ms
Enter
Sleep 1.5s

Down@400ms 3
Sleep 800ms
Type "p"
Sleep 1s
Up@400ms 2
Sleep 600ms

Type "/"
Sleep 500ms
Type@150ms "cache"
Sleep 1.5s
Escape
Sleep 800ms

Down@400ms 2
Sleep 1s
Type "q"
Sleep 1.2s
