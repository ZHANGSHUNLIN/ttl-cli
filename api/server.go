package api

import (
	"fmt"
	"net/http"
	"ttl-cli/db"
	ttlmcp "ttl-cli/mcp"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func StartServer(port int, dataDir string) error {
	userStore := db.NewUserStore(dataDir + "/users.json")
	if err := userStore.Load(); err != nil {
		return fmt.Errorf("failed to load user data: %w", err)
	}

	users := userStore.ListUsers()
	if len(users) == 0 {
		fmt.Println("Warning: No users, please run 'ttl server user add' to create users first")
	}

	tenantMgr := db.NewTenantStorageManager(dataDir + "/tenants")

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/resources", ResourcesHandler)
	mux.HandleFunc("/api/v1/resources/", ResourceHandler)
	mux.HandleFunc("/api/v1/audit/stats", AuditStatsHandler)
	mux.HandleFunc("/api/v1/history", HistoryHandler)

	mcpSrv := ttlmcp.NewTtlMCPServer()
	streamableHTTP := mcpserver.NewStreamableHTTPServer(mcpSrv)
	mux.Handle("/mcp", streamableHTTP)

	handler := MultiTenantAuthMiddleware(userStore, tenantMgr, mux)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("ttl server started, listening on %s\n", addr)
	fmt.Printf("  REST API: http://localhost:%d/api/v1/\n", port)
	fmt.Printf("  MCP HTTP: http://localhost:%d/mcp\n", port)
	fmt.Printf("  Data dir: %s\n", dataDir)
	fmt.Printf("  User count: %d\n", len(users))

	return http.ListenAndServe(addr, handler)
}
