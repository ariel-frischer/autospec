# T003 fixture scenarios

These files document deterministic fixture inputs used by jcode command/configuration tests.

- `fake-jcode.sh` records its working directory and arguments, emits fixture output, and supports a configurable non-zero exit status through environment variables.
- `missing-binary.txt` names an intentionally absent executable for missing-binary validation.
- `sdk-opt-in.yaml` represents explicit native SDK selection without starting a real SDK process.
- `unsupported-runner.yaml` represents an invalid runner value for validation tests.
