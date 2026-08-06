# Go SDK

The Go SDK is implemented in `sdk/go/` as a separate Go module:

```text
module github.com/1jehuang/jcode-go
```

## Current availability

The SDK is **in the repository source**, but it is not a globally installed command and it has not been published to the public Go module proxy. There is no `jcode-go` executable to install. Applications import the Go package, while the Jcode runtime remains a separate `jcode` executable or daemon.

The public installation command will be:

```bash
go get github.com/1jehuang/jcode-go@v0.1.0
```

Do not run that command yet. It will work only after a maintainer creates and publishes an approved release. Until then, use one of the source-checkout workflows below.

## Use from a source checkout

Clone Jcode and run the SDK checks:

```bash
git clone https://github.com/1jehuang/jcode.git
cd jcode/sdk/go
go test ./...
go vet ./...
```

To use the SDK from another local Go application, add a local replacement to that application's `go.mod`:

```text
module example.com/my-jcode-app

go 1.23

require github.com/1jehuang/jcode-go v0.0.0

replace github.com/1jehuang/jcode-go => /absolute/path/to/jcode/sdk/go
```

Then import it normally:

```go
import jcode "github.com/1jehuang/jcode-go"
```

The `replace` directive is a development workaround. Remove it and require a published semantic version after the SDK release is available.

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

The architecture, protocol, lifecycle, resilience, validation, documentation, and release-planning work is tracked under the closed `jcode-zqc` Bead epic. Publication remains a separate maintainer decision. Until a version is tagged and published, consumers should use the source-checkout replacement above rather than assuming `go get` can resolve the module.

For the lower-level API and protocol details, see [`sdk/go/README.md`](../sdk/go/README.md), [`GO_SDK_ARCHITECTURE.md`](GO_SDK_ARCHITECTURE.md), and [`GO_SDK_RELEASE_PLAN.md`](GO_SDK_RELEASE_PLAN.md).
