# Generate synthetic Claude Code session logs under a throwaway HOME, used only
# to record docs/demo.gif. None of this is real ~/.claude data. See docs/demo.tape.
import json, os, datetime, uuid

HOME = "/tmp/ccdeck-demo-home"
now = datetime.datetime.now(datetime.timezone.utc)

def ts(mins_ago):
    return (now - datetime.timedelta(minutes=mins_ago)).isoformat().replace("+00:00", "Z")

# (proj_dir, cwd, branch, title, last_prompt, model, msg_count, age_min, end_turn)
SESSIONS = [
    ("-Users-dev-acme-api",  "/Users/dev/acme-api",  "main",          "Refactor cache layer for append-only reads", "explain the cache invalidation path", "claude-opus-4-8",   42, 3,    True),
    ("-Users-dev-acme-web",  "/Users/dev/acme-web",  "feat/dark-mode","Wire up dark mode toggle",                   "make the toggle persist across reloads", "claude-sonnet-4-6", 18, 55,   False),
    ("-Users-dev-acme-api",  "/Users/dev/acme-api",  "fix/flaky-test","Investigate flaky integration test",         "why does this test pass locally but fail in CI", "claude-opus-4-8", 7, 190,  False),
    ("-Users-dev-sandbox",   "/Users/dev/sandbox",   "main",          "Add OAuth login flow",                       "add PKCE support to the login handler", "claude-sonnet-4-6", 25, 1500, False),
    ("-Users-dev-acme-web",  "/Users/dev/acme-web",  "main",          "Migrate build to pnpm workspaces",           "fix the hoisting warning on install", "claude-haiku-4-5-20251001", 11, 4300, False),
]

for proj, cwd, branch, title, prompt, model, msgs, age, end in SESSIONS:
    sid = str(uuid.uuid4())
    proj_dir = os.path.join(HOME, ".claude", "projects", proj)
    os.makedirs(proj_dir, exist_ok=True)
    path = os.path.join(proj_dir, sid + ".jsonl")
    lines = []
    lines.append({"type": "ai-title", "aiTitle": title})
    lines.append({"type": "last-prompt", "lastPrompt": prompt})
    # message turns
    for i in range(msgs):
        t = ts(age + (msgs - i) * 2)
        if i % 2 == 0:
            lines.append({"type": "user", "sessionId": sid, "cwd": cwd, "gitBranch": branch, "timestamp": t})
        else:
            stop = "end_turn" if (i == msgs - 1 and end) else "tool_use"
            lines.append({"type": "assistant", "sessionId": sid, "cwd": cwd, "gitBranch": branch,
                          "timestamp": t, "message": {"model": model, "stop_reason": stop}})
    with open(path, "w") as f:
        for l in lines:
            f.write(json.dumps(l) + "\n")
    print(f"{proj}/{sid[:8]}…  {msgs} msgs  {title}")

print("done")
