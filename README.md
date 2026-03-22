# ByteBrew Examples

Ready-to-run demos for [ByteBrew Engine](https://github.com/syntheticinc/bytebrew) -- the open-source multi-agent platform.

Each example is self-contained: clone, configure your API key, run `docker compose up`, and start chatting.

## Examples

| Example | Description | Agents | Tools |
|---------|-------------|--------|-------|
| [company-assistant](./company-assistant/) | HR + IT support multi-agent system | supervisor, hr-agent, it-support | MCP server with employee data, tickets, knowledge base |

## Quick Start

```bash
git clone https://github.com/syntheticinc/bytebrew-examples.git
cd bytebrew-examples/company-assistant

cp .env.example .env
# Edit .env -- add your OpenAI API key (or configure Ollama)

docker compose up -d
```

Then open the Admin Dashboard at [http://localhost:8443/admin](http://localhost:8443/admin) to see your agents, or start chatting via the API.

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
