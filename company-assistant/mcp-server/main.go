// Package main implements a Model Context Protocol (MCP) stdio server
// that provides company data tools: employee lookup, leave balance,
// ticket creation, and knowledge base search.
//
// Protocol: JSON-RPC 2.0 over stdin/stdout (one JSON object per line).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// --- JSON-RPC 2.0 types -----------------------------------------------------

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// --- MCP types ---------------------------------------------------------------

type toolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
}

// --- Tool definitions --------------------------------------------------------

var tools = []toolInfo{
	{
		Name:        "get_employees",
		Description: "Get a list of all employees in the company. Returns employee ID, name, email, department, title, manager, start date, and location.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "get_employee_by_id",
		Description: "Get detailed information about a specific employee by their ID (e.g. EMP001).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Employee ID (e.g. EMP001)",
				},
			},
			"required": []string{"id"},
		},
	},
	{
		Name:        "get_leave_balance",
		Description: "Get the leave balance for a specific employee, including vacation days, sick days, personal days, used days, and pending requests.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"employee_id": map[string]interface{}{
					"type":        "string",
					"description": "Employee ID (e.g. EMP001)",
				},
			},
			"required": []string{"employee_id"},
		},
	},
	{
		Name:        "create_ticket",
		Description: "Create a new IT support ticket. Returns the created ticket with its assigned ID.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Short summary of the issue",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Detailed description of the issue",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Priority level: low, medium, high, or critical",
					"enum":        []string{"low", "medium", "high", "critical"},
				},
			},
			"required": []string{"title", "description", "priority"},
		},
	},
	{
		Name:        "search_knowledge_base",
		Description: "Search the company knowledge base for articles matching a query. Covers IT guides, HR policies, and general company information.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (keywords)",
				},
			},
			"required": []string{"query"},
		},
	},
}

// --- Tool execution ----------------------------------------------------------

func executeTool(name string, args json.RawMessage) (string, error) {
	switch name {
	case "get_employees":
		return marshalResult(employees)

	case "get_employee_by_id":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("parse args: %w", err)
		}
		emp := findEmployee(p.ID)
		if emp == nil {
			return fmt.Sprintf("Employee with ID %q not found.", p.ID), nil
		}
		return marshalResult(emp)

	case "get_leave_balance":
		var p struct {
			EmployeeID string `json:"employee_id"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("parse args: %w", err)
		}
		bal, ok := leaveBalances[p.EmployeeID]
		if !ok {
			return fmt.Sprintf("No leave balance found for employee %q.", p.EmployeeID), nil
		}
		return marshalResult(bal)

	case "create_ticket":
		var p struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("parse args: %w", err)
		}
		ticket := ticketStore.Create(p.Title, p.Description, p.Priority)
		return marshalResult(ticket)

	case "search_knowledge_base":
		var p struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("parse args: %w", err)
		}
		results := searchKB(p.Query)
		if len(results) == 0 {
			return "No articles found matching your query.", nil
		}
		return marshalResult(results)

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func marshalResult(v interface{}) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// --- MCP request handlers ----------------------------------------------------

func handleInitialize(id interface{}) response {
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "company-data-mcp",
				"version": "1.0.0",
			},
		},
	}
}

func handleToolsList(id interface{}) response {
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func handleToolsCall(id interface{}, params json.RawMessage) response {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return response{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("invalid params: %v", err)},
		}
	}

	result, err := executeTool(p.Name, p.Arguments)
	if err != nil {
		return response{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: -32000, Message: err.Error()},
		}
	}

	return response{
		JSONRPC: "2.0",
		ID:      id,
		Result: toolCallResult{
			Content: []toolContent{
				{Type: "text", Text: result},
			},
		},
	}
}

// --- Main loop ---------------------------------------------------------------

func main() {
	log.SetOutput(os.Stderr) // Keep stdout clean for JSON-RPC
	log.Println("company-data-mcp server starting (stdio mode)")

	scanner := bufio.NewScanner(os.Stdin)
	// Allow up to 1 MB per line (large tool results).
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("failed to parse request: %v", err)
			continue
		}

		// Notifications (no ID) -- just acknowledge silently.
		if req.ID == nil {
			log.Printf("notification: %s", req.Method)
			continue
		}

		var resp response
		switch req.Method {
		case "initialize":
			resp = handleInitialize(req.ID)
		case "tools/list":
			resp = handleToolsList(req.ID)
		case "tools/call":
			resp = handleToolsCall(req.ID, req.Params)
		default:
			resp = response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)},
			}
		}

		if err := encoder.Encode(resp); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("stdin read error: %v", err)
	}
}
