# Jcode lifecycle recovery manual test plan

## Scope

Validate connect, private, and auto runtime policies, bounded recovery, ownership boundaries, diagnostics, and disposable smoke cleanup. Use a temporary repository and never target the developer's shared daemon for private or cleanup tests.

## Scenarios

1. **Connect with healthy shared bridge**
   - Start or identify a test-only bridge.
   - Run with `jcode.mode: connect` and a short timeout.
   - Confirm the run attaches and Autospec does not start, stop, or restart the bridge.
2. **Connect with missing bridge**
   - Use a nonexistent temporary socket.
   - Confirm prompt failure includes policy, state, attempt counts, and next action.
   - Confirm no process or socket is created.
3. **Connect after disconnect**
   - Disconnect the test bridge during a run.
   - Confirm reconnect is bounded by `reconnect_attempts` and does not replace the shared process.
4. **Private runtime recovery**
   - Run with `mode: private`, temporary `home`, and `inherit_logins: false`.
   - Force a disconnect and confirm only the run-owned runtime is restarted up to `restart_attempts`.
   - Confirm cleanup removes temporary state and leaves unrelated processes untouched.
5. **Auto shared-first and fallback**
   - With a healthy bridge, confirm `auto` selects shared ownership.
   - With the bridge absent, confirm `auto` falls back to private ownership.
6. **Startup command restriction**
   - Configure a private-only startup command in a disposable fixture.
   - Confirm it is never invoked in connect mode and failures are contextual.
7. **Disposable built-binary cheap-profile smoke**
   - Build Autospec, create a temporary repo/home/runtime/socket, and use the cheap profile.
   - Accept a validated artifact or bounded actionable failure.
   - Verify no temporary process, socket, home, or workspace residue remains.

## Report Summaries

Record actual manual results here after execution. This plan intentionally does not claim that these tests were run.

- Date:
- Tester:
- Binary/commit:
- Runtime version:
- Scenario results:
- Cleanup result:
- Diagnostics observed:
- Follow-ups:
