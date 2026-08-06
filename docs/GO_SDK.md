# Go SDK

The Go SDK is implemented in `sdk/go/` as a separate Go module:

```text
module github.com/ariel-frischer/jcode-go
```

## Current availability

The SDK is published as a separate Go module at `github.com/ariel-frischer/jcode-go` and can be consumed directly from the public Go module proxy. The Autospec integration uses the published `v0.1.1` release. There is no `jcode-go` executable to install. Applications import the Go package, while the Jcode runtime remains a separate `jcode` executable or daemon.

## Use from a source checkout

Clone Jcode and run the SDK checks:

```bash
git clone https://github.com/1jehuang/jcode.git
cd jcode/sdk/go
go test ./...
go vet ./...
```

For a standalone Go application, require the published module:

```text
module example.com/my-jcode-app

go 1.23

require github.com/ariel-frischer/jcode-go v0.1.1
```

Then import it normally:

```go
import jcode "github.com/ariel-frischer/jcode-go"
```

For SDK source development, work from the upstream Jcode repository and run its module tests directly. Autospec consumers should use the published module above rather than committing a local replacement.

## Connect to an existing Jcode runtime

The Go SDK does not start a shared runtime in connect mode. Start the local API bridge separately:

```bash
jcode api-bridge
```

Then connect from Go. The socket can be supplied explicitly, or resolved from `JCODE_API_SOCKET`, `JCODE_RUNTIME_DIR`, or `XDG_RUNTIME_DIR`:

```go
ctx := context.Background()
client, err := jcode.Connect(ctx, jcode.ConnectOptions{
    SocketPath: os.Getenv("JCODE_API_SOCKET"),
    ClientOptions: jcode.Options{
        ClientName: "my-tool/1.0",
    },
})
if err != nil {
    return err
}
defer client.Close()

session, err := client.CreateSession(ctx, jcode.CreateSessionOptions{
    WorkingDir: workingDir,
})
if err != nil {
    return err
}

stream := session.Events(ctx)
defer stream.Close()
if err := session.Send(ctx, prompt, jcode.SendOptions{}); err != nil {
    return err
}

for {
    event, err := stream.Next(ctx)
    if err != nil {
        return err
    }
    switch value := event.(type) {
    case *jcode.TextDelta:
        fmt.Print(value.Text)
    case *jcode.PermissionRequest:
        // Apply an explicit application policy before responding.
    case *jcode.TurnDone:
        return nil
    }
}
```

This mode operates on the user's shared sessions. Treat prompts, transcripts, tool calls, permissions, and file contents as sensitive.

## Launch an isolated private instance

Use `jcode.Launch` when the application should own a separate Jcode runtime:

```go
inheritLogins := false
client, err := jcode.Launch(ctx, jcode.LaunchOptions{
    Binary:        "/path/to/jcode", // optional; defaults to jcode on PATH
    WorkingDir:    workingDir,
    InheritLogins: &inheritLogins,
    ClientOptions: jcode.Options{ClientName: "private-service/1.0"},
})
if err != nil {
    return err
}
defer client.Close()
```

By default, launch creates an SDK-owned temporary home, starts an isolated bridge, waits for its API socket, and removes the temporary state when the client closes. Set `JcodeHome` to a persistent directory if sessions must survive process restarts. Persistent homes are never deleted automatically.

Credential inheritance defaults to enabled for compatibility with the existing SDKs. Set `InheritLogins` to a pointer to `false` for an empty credential store. Do not inherit a developer's credentials into untrusted or multi-tenant services.

`LaunchInstance` is the lower-level option. It starts the private process and returns its socket path without creating a client, for applications that need to own connection setup themselves.

## What is supported

| Platform | SDK build | Local runtime transport |
| --- | --- | --- |
| Linux | Supported | Unix-domain socket |
| macOS | Supported | Unix-domain socket |
| Windows | Package compile boundary | No production transport in v1 |

The protocol is harness API v1 over newline-delimited JSON. The SDK does not use Rust FFI, parse CLI terminal output, or speak ACP. Unknown event kinds are preserved for forward compatibility.

## Development and validation

From the repository:

```bash
cd sdk/go
gofmt -l .
go test ./...
go test -race ./...
go vet ./...
go mod verify
GOOS=windows GOARCH=amd64 go build ./...
```

The examples are under `sdk/go/examples/`:

- `oneshot`: connect, create a session, send one prompt, and stream output.
- `streaming`: long-lived raw event subscription.
- `private`: private-instance lifecycle pattern.

They compile without provider credentials or a live model. Runtime examples require a compatible local Jcode bridge or binary.

## Release status

The SDK is published and the Autospec module now resolves it without a relative `replace` directive. The public module repository is [`ariel-frischer/jcode-go`](https://github.com/ariel-frischer/jcode-go). The upstream Jcode runtime repository remains [`1jehuang/jcode`](https://github.com/1jehuang/jcode).

For the lower-level API and protocol details, see [`sdk/go/README.md`](../sdk/go/README.md), [`GO_SDK_ARCHITECTURE.md`](GO_SDK_ARCHITECTURE.md), and [`GO_SDK_RELEASE_PLAN.md`](GO_SDK_RELEASE_PLAN.md).
