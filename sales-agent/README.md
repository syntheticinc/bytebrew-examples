# Sales Agent Example

A ByteBrew example demonstrating a sales assistant for **TechGear Electronics** -- an AI agent that helps customers find products, check availability, create price quotes, and apply discounts.

## What This Demonstrates

### 1. Confirmation System (`confirm_before`)

Certain tool calls require explicit user approval before execution. The agent will present the action details and wait for the user to confirm or reject.

Configured tools requiring confirmation:
- `create_quote` -- creating a price quote
- `apply_discount` -- applying a discount to a quote

**Flow:**

```
User: "Create a quote for 2 ThinkPad T14s"
  |
Agent: searches products, checks inventory
  |
Agent: "I'd like to create a quote for 2x ThinkPad T14 ($1,099 each).
        Subtotal: $2,198.00 + tax. Shall I proceed?"
  |
User: "Yes, go ahead"
  |
Agent: calls create_quote -> returns QT-001
```

### 2. Settings CRUD

Business rules are stored as configurable settings that the agent reads at runtime:

| Setting | Default | Description |
|---------|---------|-------------|
| `max_discount_percent` | 15 | Maximum allowed discount on quotes |
| `free_shipping_min` | 100 | Minimum order for free shipping (USD) |
| `quote_validity_days` | 7 | Days a quote remains valid |
| `sales_tax_rate` | 8.25 | Sales tax rate (%) |

The agent checks these settings before applying discounts and creating quotes. Settings can be updated via the Engine admin API without restarting.

### 3. BYOK (Bring Your Own Key)

Clients can override the default LLM model on a per-request basis by passing headers:

```
X-Model-Provider: openai_compatible
X-Model-API-Key: sk-your-key
X-Model-Name: gpt-4o
```

The service proxy forwards these headers to the Engine, which uses them instead of the configured default model.

## Architecture

```
                        +------------------+
                        | Web Client :3000 |
                        +--------+---------+
                                 |
                          Engine REST API
                                 |
                        +--------v---------+
                        | ByteBrew Engine  |
                        | :8443            |
                        |  - Agent runtime |
                        |  - confirm_before|
                        |  - Settings API  |
                        |  - BYOK support  |
                        +--------+---------+
                                 |
                          MCP (stdio)
                                 |
                        +--------v---------+
                        |  MCP Server (Go) |
                        |  - Product catalog|
                        |  - Inventory     |
                        |  - Quotes        |
                        |  - Settings      |
                        +------------------+
```

## Quick Start

### 1. Configure

```bash
cp .env.example .env
# Edit .env -- set LLM_API_KEY at minimum
```

### 2. Start

```bash
docker compose up -d
```

This starts:
- **PostgreSQL** -- Engine database
- **MCP Server** -- Sales data tools (built as a Go binary)
- **Engine** -- ByteBrew agent runtime (port 8443)
- **Web Client** -- ByteBrew Web Client -- chat UI (port 3000)

### 3. Seed Settings

After the engine is up, seed the business rule settings:

```bash
./scripts/seed-settings.sh
```

### 4. Start chatting

Open the Web Client at [http://localhost:3000](http://localhost:3000) to chat with the sales agent.

To manage agents and settings, open the Admin Dashboard at [http://localhost:8443/admin](http://localhost:8443/admin) (login: `admin` / `changeme`).

## MCP Tools

| Tool | Description |
|------|-------------|
| `search_products` | Search catalog by keyword, category, price range |
| `check_inventory` | Check stock for a product ID |
| `create_quote` | Create a price quote (requires confirmation) |
| `apply_discount` | Apply discount to a quote (requires confirmation) |
| `get_settings` | Read a business rule setting |

## Product Catalog

12 products across 6 categories:

| Category | Products |
|----------|----------|
| Laptops | ThinkPad T14 ($1,099), Dell Latitude 5540 ($1,149), MacBook Air M4 ($1,299) |
| Monitors | Dell U2723QE ($619), LG 27UK850-W ($449) |
| Keyboards | Logitech MX Keys ($99), Keychron K2 V2 ($79) |
| Mice | Logitech MX Master 3S ($99), Logitech M720 ($49) |
| Headsets | Sony WH-1000XM5 ($349), Jabra Evolve2 75 ($299) |
| Webcams | Logitech C920 HD Pro ($69) |

## Example Conversations

### Product Search

```
User: "What laptops do you have under $1200?"

Agent: I found 2 laptops in your budget:

1. ThinkPad T14 - $1,099.00
   14" FHD, AMD Ryzen 7 PRO, 16GB RAM, 512GB SSD
   Stock: 42 available

2. Dell Latitude 5540 - $1,149.00
   15.6" FHD, Intel Core i7, 16GB RAM, 512GB SSD
   Stock: 25 available

Would you like more details or a quote?
```

### Quote with Confirmation

```
User: "Create a quote for Acme Corp: 5 ThinkPad T14s and 5 MX Keys keyboards"

Agent: I'll prepare a quote for Acme Corp:
- 5x ThinkPad T14 @ $1,099.00 = $5,495.00
- 5x Logitech MX Keys @ $99.00 = $495.00
- Subtotal: $5,990.00
- Tax (8.25%): $494.18
- Total: $6,484.18

Shall I create this quote?

User: "Yes"

Agent: Quote QT-001 created for Acme Corp.
Total: $6,484.18 (valid until 2026-03-29)
```

### Discount with Confirmation

```
User: "Apply a 10% discount to QT-001 for bulk order"

Agent: I'd like to apply a 10% discount to quote QT-001:
- Current total: $6,484.18
- After 10% discount: ~$5,888.39
- Reason: bulk order

This is within our 15% max discount policy. Shall I apply it?

User: "Go ahead"

Agent: Done! 10% discount applied to QT-001.
New total: $5,888.39 (savings: $595.79)
```

## Settings Management

Update settings via the Engine admin API:

```bash
# Increase max discount to 20%
curl -X PUT http://localhost:8443/api/v1/settings/max_discount_percent \
  -u admin:changeme \
  -H "Content-Type: application/json" \
  -d '{"value": "20", "description": "Maximum discount percentage"}'

# Change tax rate
curl -X PUT http://localhost:8443/api/v1/settings/sales_tax_rate \
  -u admin:changeme \
  -H "Content-Type: application/json" \
  -d '{"value": "10.0", "description": "Sales tax rate"}'
```

## BYOK Usage

Override the default model for a single request:

```bash
curl -X POST http://localhost:3000/api/v1/chat/sales-agent \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Model-Provider: openai_compatible" \
  -H "X-Model-API-Key: sk-your-openai-key" \
  -H "X-Model-Name: gpt-4o" \
  -d '{"message": "What monitors do you have?"}'
```

The service proxy forwards these headers to the Engine. If no BYOK headers are present, the default model from `agents.yaml` is used.

## File Structure

```
sales-agent/
  config/
    agents.yaml          # Agent + model + BYOK configuration
  mcp-server/
    main.go              # MCP stdio server (JSON-RPC 2.0)
    data.go              # Product catalog, inventory, quotes, settings
    go.mod
    Dockerfile
  service/
    main.go              # HTTP proxy: auth, rate limiting, BYOK forwarding
    go.mod
    Dockerfile
  scripts/
    seed.sh              # Import agents.yaml into Engine
    seed-settings.sh     # Create business rule settings
  docker-compose.yml
  .env.example
  README.md
```
