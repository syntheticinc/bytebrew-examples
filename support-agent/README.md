# Support Agent

A multi-agent customer support system built with [ByteBrew Engine](https://github.com/syntheticinc/bytebrew). Demonstrates **multi-agent spawn** (router delegates to specialists), **parallel tool execution**, and **MCP tool servers** with 8 support data tools.

## What It Does

A support router agent triages customer requests and spawns specialist agents to handle them. The billing specialist handles subscriptions, invoices, and refunds. The technical specialist runs parallel diagnostics to identify and resolve issues.

```
                         +-------------------+
                         |  Support Router   |
     User request -----> |                   |
                         |  - get_customer   |
                         |  - ask_user       |
                         +--------+----------+
                                  |
                     spawn        |        spawn
               +------------------+------------------+
               |                                     |
    +----------v----------+            +-------------v-----------+
    |  Billing Specialist |            |  Technical Specialist   |
    |                     |            |                         |
    |  - get_customer     |            |  - check_service_status |
    |  - get_ticket       |            |  - get_error_logs       |
    |  - create_ticket    |            |  - search_kb            |
    |  - search_kb        |            |  - get_customer         |
    |  - update_sub       |            |  - get_ticket           |
    |  - process_refund   |            |  - create_ticket        |
    +---------------------+            +-------------------------+
                                              parallel execution
```

## Key Features Demonstrated

### 1. Multi-Agent Spawn

The router agent analyzes incoming requests and spawns the appropriate specialist. This is configured via the `can_spawn` field in `agents.yaml`:

```yaml
agents:
  - name: "support-router"
    can_spawn: [billing, technical]    # Can spawn these agents
    ...

  - name: "billing"
    lifecycle: spawn                   # Only created when spawned
    ...

  - name: "technical"
    lifecycle: spawn                   # Only created when spawned
    tool_execution: parallel           # Runs tools concurrently
    ...
```

When the router calls `spawn("billing", "Customer CUST-003 was overcharged...")`, the engine creates a new billing agent instance with its own context and tools.

### 2. Parallel Tool Execution

The technical specialist is configured with `tool_execution: parallel`. When the LLM requests multiple tool calls in a single step (e.g., `check_service_status("storage")` + `get_error_logs("CUST-001")` + `search_kb("file sync timeout")`), the engine executes them concurrently instead of sequentially. This reduces diagnostic time significantly.

### 3. MCP Tool Server (8 Tools)

A Go-based MCP stdio server provides customer support data tools with realistic mock data:

| Tool | Description |
|------|-------------|
| `get_customer` | Look up customer by ID, email, or name |
| `get_ticket` | Get support ticket details |
| `create_ticket` | Create a new support ticket |
| `search_kb` | Search knowledge base articles |
| `check_service_status` | Check CloudSync service health |
| `get_error_logs` | Get customer error logs |
| `update_subscription` | Change customer plan |
| `process_refund` | Issue refund (>$100 needs approval) |

**Mock data includes:**
- 8 customers on Starter/Pro/Enterprise plans
- 15 existing support tickets (technical + billing)
- 5 knowledge base articles (sync, API, billing, SSO, webhooks)
- 5 microservices (1 degraded: storage)
- Realistic error logs per customer
- 9 invoices with payment status

## Example Conversations

### Billing Issue

> **User:** "Hi, I'm Carol Martinez. I was charged $29 instead of $19 on my last invoice."
>
> Router looks up Carol (CUST-003, Starter plan at $19/mo), sees the billing discrepancy, and spawns the **billing** specialist. The billing specialist finds ticket TKT-003 already open about this issue, verifies the overcharge on invoice INV-2026-003 ($29 vs expected $19), and processes a $10 refund.

### Technical Issue

> **User:** "Our large file uploads keep timing out. Customer ID is CUST-001."
>
> Router looks up CUST-001 (Alice Chen, TechStartup Inc, Pro plan) and spawns the **technical** specialist. The technical specialist runs diagnostics **in parallel**: checks storage service status (degraded, elevated latency), pulls error logs (upload timeouts for 500MB+ files), and searches KB (finds chunked upload solution). Provides diagnosis: storage service is degraded + recommends enabling chunked uploads.

### Mixed Issue

> **User:** "We need to upgrade from Pro to Enterprise and we're also having SSO issues. Customer: FinServ Solutions."
>
> Router looks up FinServ Solutions (CUST-004, already Enterprise), spawns **both** specialists. The billing specialist clarifies they're already on Enterprise. The technical specialist checks auth-service, finds SAML certificate mismatch errors, references KB article on SSO troubleshooting, and provides resolution steps.

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
cd bytebrew-examples/support-agent

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
| **db** | PostgreSQL (pgvector) -- agent state and knowledge storage |
| **mcp-server** | Builds the support data MCP tool server binary |
| **engine** | ByteBrew Engine -- the AI agent platform (port 8443) |
| **web-client** | ByteBrew Web Client -- chat UI (port 3000) |

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

### 4. Start chatting

Open the Web Client at [http://localhost:3000](http://localhost:3000) to chat with the support agent.

To manage agents, open the Admin Dashboard at [http://localhost:8443/admin](http://localhost:8443/admin) and log in with:
- Username: `admin` (or whatever you set in `.env`)
- Password: `changeme` (or whatever you set in `.env`)

You should see three agents: **support-router**, **billing**, and **technical**.

## Chat via the API

### Create a session and send a message

```bash
# Create a session
SESSION=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  http://localhost:8443/api/v1/sessions \
  | jq -r '.id')

echo "Session ID: $SESSION"

# Send a billing request
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "I am Carol Martinez and I was overcharged on my last invoice. Can you help?"}' \
  http://localhost:8443/api/v1/sessions/$SESSION/messages \
  | jq .
```

### Stream events via SSE

```bash
curl -N http://localhost:8443/api/v1/sessions/$SESSION/events
```

### Technical support request

```bash
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"message": "Our file uploads keep timing out. We are TechStartup Inc (CUST-001). Files over 500MB fail every time."}' \
  http://localhost:8443/api/v1/sessions/$SESSION/messages \
  | jq .
```

## How Spawn Works

When the router determines a request needs a specialist:

1. **Router analyzes** the request and calls the built-in `spawn` tool
2. **Engine creates** a new agent instance from the spawned agent's config
3. **Spawned agent** receives the task description as its initial message
4. **Spawned agent** uses its own tools to resolve the issue
5. **Result flows back** to the router, which summarizes for the user

The spawned agent has its own step counter, tool access, and conversation history. It runs independently and returns its result when complete.

```
Router step 1: get_customer("Carol Martinez")
Router step 2: spawn("billing", "CUST-003 was charged $29 instead of $19...")
  Billing step 1: get_ticket("TKT-003")
  Billing step 2: process_refund("INV-2026-003", 1000, "overcharge")
  Billing step 3: [returns result to router]
Router step 3: [summarizes billing result for user]
```

## Project Structure

```
support-agent/
├── docker-compose.yml              # Engine + PostgreSQL + MCP server
├── .env.example                    # Environment variable template
├── config/
│   └── agents.yaml                 # 3 agents: router, billing, technical
├── scripts/
│   └── seed-config.sh              # Auto-import config on first startup
├── service/
│   ├── Dockerfile                  # Optional proxy service
│   ├── go.mod
│   └── main.go                     # JWT auth, rate limiting, SSE proxy
└── mcp-server/
    ├── Dockerfile                  # Builds the MCP server binary
    ├── go.mod
    ├── main.go                     # MCP stdio server (JSON-RPC 2.0, 8 tools)
    └── data.go                     # Mock support data (8 customers, 15 tickets)
```

## Customizing

- **Add more customers:** Edit `mcp-server/data.go`, rebuild with `docker compose build mcp-server`
- **Change routing logic:** Edit the router's system prompt in `config/agents.yaml`
- **Add a new specialist:** Add a new agent with `lifecycle: spawn` in `agents.yaml` and add it to the router's `can_spawn` list
- **Disable parallel execution:** Remove `tool_execution: parallel` from the technical agent
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

**Agent not spawning specialists:**
- Verify config was imported: check the Admin Dashboard for all 3 agents
- Check that `can_spawn` lists the correct agent names
- Check engine logs: `docker compose logs -f engine`

**Parallel execution not working:**
- Ensure `tool_execution: parallel` is set in agents.yaml
- The LLM must request multiple tool calls in a single response for parallel execution to activate
- Not all LLMs support multi-tool-call responses

**MCP server errors:**
- Rebuild: `docker compose build mcp-server`
- Check if binary exists: `docker compose exec engine ls -la /opt/mcp/`
