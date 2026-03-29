# Widget Integration Example

Demonstrates embedding the ByteBrew AI chat widget on a website.

## Prerequisites

- ByteBrew Engine running (`docker compose up -d`)
- An agent configured and marked as **Public**
- An API key with **Chat (Public)** scope

## Quick Start

1. Edit `index.html` — replace `YOUR_AGENT_NAME` and `YOUR_API_KEY`
2. Open `index.html` in your browser
3. Click the chat bubble in the bottom-right corner
4. Send a message — the response streams in real-time

## Configuration

| Attribute | Required | Description |
|-----------|----------|-------------|
| `data-agent` | Yes | Agent name to chat with |
| `data-api-key` | Yes* | API key for authentication |
| `data-endpoint` | No | Custom backend URL (proxy mode) |
| `data-theme` | No | `light` (default) or `dark` |
| `data-position` | No | `bottom-right` (default) or `bottom-left` |
| `data-title` | No | Chat panel title (default: "Chat") |

*Either `data-api-key` (direct mode) or `data-endpoint` (proxy mode) is required.

## Modes

### Direct Mode
Widget connects directly to ByteBrew Engine. Use for simple integrations where the API key can be public.

### Proxy Mode
Widget connects to your backend, which forwards requests to ByteBrew Engine. Use when you need to add authentication, rate limiting, or user context.

```html
<script src="http://localhost:8443/widget.js"
        data-agent="support-bot"
        data-endpoint="https://myapp.com/api/ai-chat">
</script>
```
