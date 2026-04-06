package mcp

import (
	"ttl-cli/models"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewTtlMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"ttl",
		models.Version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	registerAllTools(s)
	return s
}

func registerAllTools(s *server.MCPServer) {
	s.AddTools(
		server.ServerTool{Tool: addTool, Handler: handleAdd},
		server.ServerTool{Tool: getTool, Handler: handleGet},
		server.ServerTool{Tool: updateTool, Handler: handleUpdate},
		server.ServerTool{Tool: deleteTool, Handler: handleDelete},
		server.ServerTool{Tool: tagTool, Handler: handleTag},
		server.ServerTool{Tool: dtagTool, Handler: handleDtag},
		server.ServerTool{Tool: renameTool, Handler: handleRename},
		server.ServerTool{Tool: listTool, Handler: handleList},
		server.ServerTool{Tool: logAddTool, Handler: handleLogAdd},
		server.ServerTool{Tool: logListTool, Handler: handleLogList},
		server.ServerTool{Tool: logDeleteTool, Handler: handleLogDelete},
		server.ServerTool{Tool: exportTool, Handler: handleExport},
	)
}

var addTool = mcp.NewTool("ttl_add",
	mcp.WithDescription("Add a new resource (key must not already exist)"),
	mcp.WithString("key", mcp.Required(), mcp.Description("Unique resource name")),
	mcp.WithString("value", mcp.Required(), mcp.Description("Resource content, supports escape sequences like \\n \\t")),
)

var getTool = mcp.NewTool("ttl_get",
	mcp.WithDescription("Fuzzy search resources (matches key and tag), omit key to list all"),
	mcp.WithString("key", mcp.Description("Search keyword, fuzzy matches resource name and tags")),
)

var updateTool = mcp.NewTool("ttl_update",
	mcp.WithDescription("Update resource content (keeps existing tags)"),
	mcp.WithString("key", mcp.Required(), mcp.Description("Unique resource name (exact match)")),
	mcp.WithString("value", mcp.Required(), mcp.Description("New resource content")),
)

var deleteTool = mcp.NewTool("ttl_delete",
	mcp.WithDescription("Delete resource (also cleans up audit and history records)"),
	mcp.WithString("key", mcp.Required(), mcp.Description("Unique resource name (exact match)")),
)

var tagTool = mcp.NewTool("ttl_tag",
	mcp.WithDescription("Add one or more tags to a resource"),
	mcp.WithString("key", mcp.Required(), mcp.Description("Unique resource name")),
	mcp.WithArray("tags", mcp.Required(), mcp.Description("List of tags to add"), mcp.WithStringItems()),
)

var dtagTool = mcp.NewTool("ttl_dtag",
	mcp.WithDescription("Remove a tag from a resource"),
	mcp.WithString("key", mcp.Required(), mcp.Description("Unique resource name")),
	mcp.WithString("tag", mcp.Required(), mcp.Description("Tag to remove")),
)

var renameTool = mcp.NewTool("ttl_rename",
	mcp.WithDescription("Rename a resource"),
	mcp.WithString("old_key", mcp.Required(), mcp.Description("Current resource name")),
	mcp.WithString("new_key", mcp.Required(), mcp.Description("New resource name")),
)

var listTool = mcp.NewTool("ttl_list",
	mcp.WithDescription("List all resource keys"),
)

var logAddTool = mcp.NewTool("ttl_log_add",
	mcp.WithDescription("Write a work log entry"),
	mcp.WithString("content", mcp.Required(), mcp.Description("Log content")),
	mcp.WithArray("tags", mcp.Description("Category tags, e.g. [\"projectA\", \"review\"]"), mcp.WithStringItems()),
)

var logListTool = mcp.NewTool("ttl_log_list",
	mcp.WithDescription("View work logs, supports date/tag filtering"),
	mcp.WithString("date", mcp.Description("Specific date (YYYY-MM-DD), default today")),
	mcp.WithBoolean("week", mcp.Description("View this week's logs (Monday to today)")),
	mcp.WithBoolean("month", mcp.Description("View this month's logs (1st to today)")),
	mcp.WithString("from", mcp.Description("Range start date (YYYY-MM-DD)")),
	mcp.WithString("to", mcp.Description("Range end date (YYYY-MM-DD)")),
	mcp.WithString("tag", mcp.Description("Filter by tag, only show logs containing this tag")),
)

var logDeleteTool = mcp.NewTool("ttl_log_delete",
	mcp.WithDescription("Delete a work log entry"),
	mcp.WithNumber("id", mcp.Required(), mcp.Description("Log ID (UnixNano timestamp)")),
)

var exportTool = mcp.NewTool("ttl_export",
	mcp.WithDescription("Export data as CSV format"),
	mcp.WithString("type", mcp.Description("Export content type: resources / audit / history / log, default resources")),
	mcp.WithBoolean("bom", mcp.Description("Write UTF-8 BOM (fixes Excel Chinese garbled text), default false")),
)
