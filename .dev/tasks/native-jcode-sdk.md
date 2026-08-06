# Native jcode SDK integration manual test plan

## Prerequisites

- Build autospec from this checkout.
- Have a compatible `jcode` checkout/runtime available separately.
- Do not modify the sibling jcode repository during testing.

## Connect mode

1. Start an existing runtime with `jcode api-bridge`.
2. Configure `agent_preset: jcode` and `jcode.mode: connect`.
3. Run a small workflow with `autospec specify` or `autospec run`.
4. Confirm TextDelta output is streamed until TurnDone and the command exits successfully.
5. Confirm an unavailable socket produces a contextual connection error.

## Private mode

1. Configure `agent_preset: jcode`, `jcode.mode: private`, and an explicit binary if needed.
2. Run a small workflow in a temporary project.
3. Confirm autospec starts an isolated runtime, streams output, and cleans up its SDK-owned temporary home.
4. Repeat with `inherit_logins: false` and verify the private runtime does not reuse shared credentials.
5. Interrupt the workflow and confirm cancellation returns promptly and runtime cleanup is attempted.

## Compatibility and safety

1. Run workflows with `claude`, `codex`, and `opencode` presets to confirm existing agents are unchanged.
2. Emit permission and unknown events from a compatible test runtime and confirm the stream remains safe and continues to TurnDone.
3. Run `autospec doctor` and confirm jcode is listed without requiring a CLI exec runner.

## Report Summaries

- Connect mode:
- Private mode:
- Cancellation and cleanup:
- Permission/unknown event handling:
- Existing-agent regression checks:
- Tester/date:
