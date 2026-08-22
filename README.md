# Kiro plugin for CLIProxyAPI

Native Windows plugin that connects CLIProxyAPI 7.2.x to a Kiro subscription through AWS IAM Identity Center.

The plugin:

- discovers the Kiro profile with the access token of each account;
- keeps Identity Center registrations isolated;
- loads only the models available to the authenticated account;
- derives reasoning effort levels and the upstream field path from each model's live Kiro schema;
- exposes upstream model IDs without a `kiro/` prefix;
- supports OpenAI Chat Completions, OpenAI Responses, and Anthropic Messages;
- supports streaming, tool calls, token refresh, multiple accounts, and failover;
- provides a read-only `Kiro Usage` page for subscription usage.

`Kiro` is the name shown in the CLIProxyAPI interface. The stable provider ID, configuration key, and DLL filename use `kiro`.

## Requirements

- CLIProxyAPI 7.2.138
- Windows amd64
- Go 1.26
- A C compiler available to CGO
- A Kiro account connected through AWS IAM Identity Center

## Build

```powershell
go test ./...
go vet ./...
New-Item -ItemType Directory -Force dist | Out-Null
go build -trimpath -buildmode=c-shared -o dist/kiro.dll ./cmd/kiro-plugin
$hash = (Get-FileHash dist/kiro.dll -Algorithm SHA256).Hash.ToLowerInvariant()
"$hash  kiro.dll" | Set-Content -NoNewline -Encoding ascii dist/kiro.dll.sha256
```

The GitHub Actions workflow runs the same tests and publishes `kiro.dll`, `kiro.h`, and `kiro.dll.sha256`. Tagged builds are attached to the matching GitHub release.

## Install

Stop CLIProxyAPI before replacing the DLL. Back up the existing DLL and configuration, then copy `dist/kiro.dll` to the configured plugin directory.

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    kiro:
      enabled: true
      priority: 1
      auth_method: idc
```

Restart CLIProxyAPI and open the Kiro login action in the Management Center. Enter the AWS IAM Identity Center Start URL and region for that account. The form does not provide default values.

The plugin stores the resulting Identity Center session through the CLIProxyAPI authentication mechanism. Do not put passwords, management keys, access tokens, refresh tokens, or client secrets in the configuration or repository.

## Model capabilities

Reasoning controls are advertised only when the authenticated account's Kiro model schema declares an `effort` enum. Claude models currently use `additionalModelRequestFields.output_config.effort`; GPT models use `additionalModelRequestFields.reasoning.effort`. The plugin forwards the selected level through that declared path for OpenAI Responses, Chat Completions, and Anthropic Messages.

The loopback resource `/v0/resource/plugins/kiro/capabilities` exposes only the intersection of non-secret model capability metadata discovered for the connected accounts. Local catalog synchronizers can use it instead of maintaining guessed model lists. It contains no account identifiers, profile ARNs, tokens, or quota data.

## Kiro Usage

Open `Kiro Usage` from the plugin menu in the Management Center. The plugin renders one card per connected Kiro account with the plan, usage buckets, balance, renewal date, and overage information returned by Kiro.

The page is read-only and contains no JavaScript. Its 192-bit random route is generated when CLIProxyAPI starts and is revealed only through the authenticated plugin list. The route changes after a process restart. Results remain in memory for 60 seconds, and a manual refresh is limited to one upstream call per account every 10 seconds.

Quota data and credentials are never written by the page. If Kiro changes or rejects its private usage endpoint, the affected account displays an error instead of an estimated value.

## Architecture

This is a standalone Go module built against the public CLIProxyAPI v7 plugin SDK. The repository contains only the Kiro provider: IAM Identity Center authentication, model discovery, request and response translation, execution, and the read-only usage page. It does not embed the CLIProxyAPI server or unrelated providers.

The plugin is distributed under the [MIT License](LICENSE).
