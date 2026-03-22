# ByteBrew Examples

Ready-to-run demos for [ByteBrew Engine](https://github.com/syntheticinc/bytebrew) -- the open-source multi-agent platform.

Each example is self-contained: clone, configure your API key, run `docker compose up`, and start chatting.

## Examples

| Example | Description | Agents | Key Features |
|---------|-------------|--------|--------------|
| [hr-assistant](./hr-assistant/) | AI-powered HR assistant with leave management and company policies | hr-assistant | Knowledge Base (RAG), ask_user, MCP tools, escalation |
| [support-agent](./support-agent/) | Multi-agent customer support with specialist routing | support-router, billing, technical | Multi-agent spawn, parallel tool execution, 8 MCP tools |
| [sales-agent](./sales-agent/) | Sales assistant for product search, quotes, and discounts | sales-agent | confirm_before, Settings CRUD, BYOK (Bring Your Own Key) |

## Quick Start

```bash
git clone https://github.com/syntheticinc/bytebrew-examples.git
cd bytebrew-examples/hr-assistant

cp .env.example .env
# Edit .env -- add your LLM API key (OpenAI, OpenRouter, or configure Ollama)

docker compose up -d
```

Open the Web Client at [http://localhost:3000](http://localhost:3000) to start chatting, or the Admin Dashboard at [http://localhost:8443/admin](http://localhost:8443/admin) to manage your agents.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- An LLM API key (OpenAI, or any OpenAI-compatible provider) **OR** [Ollama](https://ollama.ai/) running locally

## Contributing

1. Fork this repository
2. Create a new directory for your example (e.g. `my-example/`)
3. Include a `README.md` with setup instructions and a `docker-compose.yml`
4. Submit a pull request

Each example should be fully self-contained and runnable with `docker compose up`.

## License

MIT
