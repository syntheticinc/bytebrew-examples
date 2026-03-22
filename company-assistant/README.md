# Company Assistant

A multi-agent company assistant built with [ByteBrew Engine](https://github.com/syntheticinc/bytebrew). Demonstrates agent routing, MCP tool servers, and multi-agent collaboration.

## What It Does

Three agents work together to handle employee requests:

```
                         +------------------+
                         |   Supervisor     |
   User request -------> |  (routes to the  |
                         |  right agent)    |
                         +--------+---------+
                                  |
                    +-------------+-------------+
                    |                           |
           +--------v--------+        +--------v--------+
           |    HR Agent     |        |   IT Support    |
           |                 |        |                 |
           | - get_employees |        | - create_ticket |
           | - leave_balance |        | - search_kb     |
           | - search_kb     |        |                 |
           +-----------------+        +-----------------+
```

**Example conversations:**

> **User:** "How many vacation days does Alice Johnson have left?"
>
> Supervisor routes to HR Agent, which calls `get_leave_balance` and responds with the balance details.

> **User:** "My VPN is not connecting, can you help?"
>
> Supervisor routes to IT Support, which searches the knowledge base for VPN troubleshooting steps.

> **User:** "I need a new monitor for my home office."
>
> Supervisor routes to IT Support, which creates a ticket and explains the approval process.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- An LLM API key (**one** of):
  - [OpenAI API key](https://platform.openai.com/api-keys) (recommended: `gpt-4o`)
  - Any OpenAI-compatible API (Groq, Together, etc.)
  - [Ollama](https://ollama.ai/) running locally (free, no API key needed)

## Setup

### 1. Clone and configure

```bash
git clone https://github.com/syntheticinc/bytebrew-examples.git
cd bytebrew-examples/company-assistant

cp .env.example .env
```

Edit `.env` and add your API key:

```env
LLM_API_KEY=sk-your-openai-key-here
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=gpt-4o
```

**Using Ollama instead?** Make sure Ollama is running (`ollama serve`), then:

```env
LLM_API_KEY=ollama
LLM_BASE_URL=http://host.docker.internal:11434/v1
LLM_MODEL=llama3.2
```

### 2. Start the stack

```bash
docker compose up -d
```

This starts three services:
- **PostgreSQL** (pgvector) -- agent state and configuration storage
- **MCP Server** -- builds the company data tool server binary
- **ByteBrew Engine** -- the multi-agent platform (port 8443)

Wait ~30 seconds for the engine to start and import the agent configuration.

### 3. Import agent configuration

On first startup, import the agent config into the engine:

```bash
curl -X POST \
  -H "Content-Type: application/x-yaml" \
  -u admin:changeme \
  -d @config/agents.yaml \
  http://localhost:8443/api/v1/config/import
```

### 4. Verify in the Admin Dashboard

Open [http://localhost:8443/admin](http://localhost:8443/admin) and log in with:
- Username: `admin` (or whatever you set in `.env`)
- Password: `changeme` (or whatever you set in `.env`)

You should see three agents: **supervisor**, **hr-agent**, and **it-support**.

## Chat via the API

### Create a session and send a message

```bash
# Create a session
curl -s -X POST \
  -H "Content-Type: application/json" \
  http://localhost:8443/api/v1/sessions \
  | jq .

# Send a message (replace SESSION_ID with the ID from above)
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "How many vacation days does Alice Johnson have?"}' \
  http://localhost:8443/api/v1/sessions/SESSION_ID/messages \
  | jq .
```

### Stream events via SSE

```bash
curl -N http://localhost:8443/api/v1/sessions/SESSION_ID/events
```

## Chat via ByteBrew Web Client

For a full chat UI, use the [ByteBrew Web Client](https://github.com/syntheticinc/bytebrew-web-client):

```bash
# In a separate directory
git clone https://github.com/syntheticinc/bytebrew-web-client.git
cd bytebrew-web-client
npm install
VITE_ENGINE_URL=http://localhost:8443 npm run dev
```

Then open [http://localhost:5173](http://localhost:5173) in your browser.

## Project Structure

```
company-assistant/
├── docker-compose.yml          # Engine + PostgreSQL + MCP server
├── .env.example                # Environment variable template
├── config/
│   └── agents.yaml             # Agent definitions (supervisor, hr, it-support)
├── scripts/
│   └── seed-config.sh          # Auto-import config on first startup
└── mcp-server/
    ├── Dockerfile              # Builds the MCP server binary
    ├── go.mod
    ├── main.go                 # MCP stdio server (JSON-RPC 2.0)
    └── data.go                 # Mock data (employees, tickets, KB)
```

## How It Works

### Agent Routing (Supervisor)

The supervisor agent receives all user messages. Based on the content, it spawns either `hr-agent` or `it-support` to handle the request. This is configured via `can_spawn` in the agent definition.

### MCP Tool Server

The `hr-agent` and `it-support` agents connect to the same MCP server, which provides five tools:

| Tool | Description |
|------|-------------|
| `get_employees` | List all employees |
| `get_employee_by_id` | Get details for one employee |
| `get_leave_balance` | Check vacation/sick/personal days |
| `create_ticket` | Create an IT support ticket |
| `search_knowledge_base` | Search HR policies and IT guides |

The MCP server communicates via **stdio** (JSON-RPC 2.0 over stdin/stdout). The engine spawns it as a subprocess and sends tool calls as JSON-RPC requests.

### Customizing

- **Add more employees:** Edit `mcp-server/data.go` and rebuild
- **Change agent behavior:** Edit `config/agents.yaml` and re-import
- **Add new tools:** Add a tool definition in `main.go` and handler in `executeTool()`
- **Use a different LLM:** Change `LLM_BASE_URL` and `LLM_MODEL` in `.env`

## Stopping

```bash
docker compose down        # Stop containers (keep data)
docker compose down -v     # Stop and delete all data
```

## Troubleshooting

**Engine won't start:**
Check logs with `docker compose logs engine`. Common issues:
- Database not ready yet (wait a few seconds and try again)
- Invalid API key or base URL

**Agent not responding:**
- Verify config was imported: check the Admin Dashboard
- Check engine logs for LLM errors: `docker compose logs -f engine`

**MCP server errors:**
- Rebuild: `docker compose build mcp-server`
- Check if binary exists: `docker compose exec engine ls -la /opt/mcp/`
