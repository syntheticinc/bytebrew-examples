# HR Assistant

An AI-powered HR assistant built with [ByteBrew Engine](https://github.com/syntheticinc/bytebrew). Demonstrates **Knowledge Base (RAG)**, **ask_user** interactive flow, **MCP tool servers**, and **escalation** to human agents.

## What It Does

A single HR assistant agent helps employees with leave management, benefits questions, onboarding information, and company policies. It combines a knowledge base (5 company handbook documents) with live HR data tools via MCP.

```
                    +---------------------------+
                    |      HR Assistant         |
  User request ---> |                           |
                    |  Built-in tools:          |
                    |  - knowledge_search (RAG) |
                    |  - ask_user               |
                    |  - manage_tasks           |
                    |                           |
                    |  MCP tools (hr-data):     |
                    |  - get_employee           |
                    |  - get_leave_balance      |
                    |  - submit_leave_request   |
                    +---------------------------+
                              |
                    (escalation webhook)
                              |
                    +---------------------------+
                    |   Escalation Service      |
                    |   (placeholder)           |
                    +---------------------------+
```

## Key Features Demonstrated

### 1. Knowledge Base (RAG)
The `config/knowledge/` directory contains 5 company handbook documents that the engine indexes automatically. The agent uses `knowledge_search` to find relevant policies before answering questions.

**Documents included:**
- `pto-policy.md` -- PTO accrual rates, carryover rules, blackout periods, holidays
- `benefits.md` -- Health/dental/vision insurance, 401(k), life insurance, FSA/HSA
- `onboarding.md` -- New hire checklist, equipment policy, 90-day milestones
- `remote-work.md` -- Hybrid model, core hours, home office stipend, travel
- `code-of-conduct.md` -- Professional behavior, anti-harassment, reporting, social media

### 2. ask_user (Interactive Flow)
When submitting a leave request, the agent uses `ask_user` to interactively collect required information (dates, leave type, reason) from the employee before calling the MCP tool.

### 3. MCP Tool Server
A Go-based MCP stdio server provides 3 HR data tools with mock data for 10 employees:

| Tool | Description |
|------|-------------|
| `get_employee` | Look up employee by ID, email, or name |
| `get_leave_balance` | Check vacation/sick/personal day balances |
| `submit_leave_request` | Submit a leave request with validation |

### 4. Escalation
When the agent cannot resolve an issue, it triggers an escalation webhook to hand off to a human HR specialist.

## Example Conversations

> **User:** "How many vacation days do I get per year?"
>
> Agent searches knowledge base, finds `pto-policy.md`, and explains the accrual tiers (15/20/25 days based on tenure).

> **User:** "What's Alice Johnson's leave balance?"
>
> Agent calls `get_employee` to find Alice (EMP001), then `get_leave_balance` to show remaining days.

> **User:** "I'd like to request time off next week."
>
> Agent uses `ask_user` to collect start date, end date, leave type, and reason, then calls `submit_leave_request`.

> **User:** "I'm dealing with a complex situation with my manager."
>
> Agent recognizes this as an escalation trigger and offers to connect the employee with a human HR specialist.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- An LLM API key (**one** of):
  - [OpenRouter](https://openrouter.ai/) (recommended -- access to many models)
  - [OpenAI API key](https://platform.openai.com/api-keys) (recommended: `gpt-4o`)
  - Any OpenAI-compatible API (Groq, Together, etc.)
  - [Ollama](https://ollama.ai/) running locally (free, no API key needed)

## Setup

### 1. Clone and configure

```bash
git clone https://github.com/syntheticinc/bytebrew-examples.git
cd bytebrew-examples/hr-assistant

cp .env.example .env
```

Edit `.env` and add your API key:

```env
# OpenRouter (default)
LLM_API_KEY=sk-or-your-openrouter-key
LLM_BASE_URL=https://openrouter.ai/api/v1
LLM_MODEL=qwen/qwen3-coder-next
```

**Using OpenAI instead?**

```env
LLM_API_KEY=sk-your-openai-key-here
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=gpt-4o
```

**Using Ollama?** Make sure Ollama is running (`ollama serve`), then:

```env
LLM_API_KEY=ollama
LLM_BASE_URL=http://host.docker.internal:11434/v1
LLM_MODEL=llama3.2
```

### 2. Start the stack

```bash
docker compose up -d
```

This starts four services:

| Service | Description |
|---------|-------------|
| **db** | PostgreSQL (pgvector) -- agent state and knowledge base storage |
| **mcp-server** | Builds the HR data MCP tool server binary |
| **engine** | ByteBrew Engine -- the AI agent platform (port 8443) |
| **service** | Placeholder escalation webhook receiver (port 3000) |

Wait ~30 seconds for the engine to start and import the agent configuration.

### 3. Import agent configuration

On first startup, the seed script runs automatically. If you need to re-import:

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

You should see the **hr-assistant** agent with its tools and knowledge base configured.

## Chat via the API

### Create a session and send a message

```bash
# Create a session
SESSION=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  http://localhost:8443/api/v1/sessions \
  | jq -r '.id')

echo "Session ID: $SESSION"

# Send a message
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "How many vacation days does Alice Johnson have left?"}' \
  http://localhost:8443/api/v1/sessions/$SESSION/messages \
  | jq .
```

### Stream events via SSE

```bash
curl -N http://localhost:8443/api/v1/sessions/$SESSION/events
```

### Interactive leave request

```bash
# Start the conversation
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "I would like to request vacation time. I am Alice Johnson (EMP001)."}' \
  http://localhost:8443/api/v1/sessions/$SESSION/messages \
  | jq .

# The agent will ask for dates, type, and reason via ask_user.
# Respond to the follow-up questions:
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "April 14 to April 18, vacation, family trip to Hawaii"}' \
  http://localhost:8443/api/v1/sessions/$SESSION/messages \
  | jq .
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
hr-assistant/
├── docker-compose.yml              # Engine + PostgreSQL + MCP + service
├── .env.example                    # Environment variable template
├── config/
│   ├── agents.yaml                 # Agent definition (hr-assistant)
│   └── knowledge/                  # Company handbook (indexed for RAG)
│       ├── pto-policy.md           # PTO / vacation policy
│       ├── benefits.md             # Health, dental, 401(k), etc.
│       ├── onboarding.md           # New hire checklist
│       ├── remote-work.md          # Hybrid & remote work policy
│       └── code-of-conduct.md      # Workplace conduct guidelines
├── scripts/
│   └── seed.sh                     # Auto-import config on first startup
└── mcp-server/
    ├── Dockerfile                  # Builds the MCP server binary
    ├── go.mod
    ├── main.go                     # MCP stdio server (JSON-RPC 2.0)
    └── data.go                     # Mock HR data (10 employees)
```

## How It Works

### Knowledge Base (RAG)
The engine indexes all markdown files in `config/knowledge/` on startup. When the agent calls `knowledge_search`, the engine performs semantic search over these documents and returns relevant passages. The agent then uses these passages to formulate accurate, policy-backed answers.

### MCP Tool Server
The MCP server runs as a subprocess spawned by the engine. Communication uses JSON-RPC 2.0 over stdin/stdout (stdio transport). The server provides three tools with mock data for 10 employees across 5 departments.

**Employee data includes:**
- Engineering: Alice Johnson, Bob Martinez, David Kim, Henry Wilson
- Marketing: Carol Williams
- HR: Emily Davis
- Sales: Frank Thompson, Lisa Park
- Finance: Grace Lee, Michael Brown

### ask_user Flow
When the agent needs information from the user (e.g., dates for a leave request), it uses the `ask_user` built-in tool. This sends a question back to the user and waits for their response before proceeding. This creates a natural conversational flow.

### Escalation
The agent configuration includes escalation triggers. When the user mentions keywords like "escalate", "need human", or "complex situation", the engine triggers the escalation webhook (`POST /webhooks/escalation` on the service container). In production, this would notify a human HR specialist.

## Customizing

- **Add more employees:** Edit `mcp-server/data.go`, rebuild with `docker compose build mcp-server`
- **Update policies:** Edit files in `config/knowledge/`, restart engine to re-index
- **Change agent behavior:** Edit `config/agents.yaml` and re-import
- **Add new MCP tools:** Add tool definitions in `main.go` and handlers in `data.go`
- **Use a different LLM:** Change `LLM_BASE_URL` and `LLM_MODEL` in `.env`
- **Connect real escalation:** Replace the placeholder service with your ticketing system

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

**Knowledge search returns no results:**
- Verify knowledge files are mounted: `docker compose exec engine ls /etc/bytebrew/knowledge/`
- Check engine logs for indexing errors: `docker compose logs -f engine`

**Agent not responding:**
- Verify config was imported: check the Admin Dashboard
- Check engine logs for LLM errors: `docker compose logs -f engine`

**MCP server errors:**
- Rebuild: `docker compose build mcp-server`
- Check if binary exists: `docker compose exec engine ls -la /opt/mcp/`

**Leave request validation fails:**
- Start dates must be today or in the future
- Employee must have enough balance for the leave type
- Date range must contain at least one weekday
