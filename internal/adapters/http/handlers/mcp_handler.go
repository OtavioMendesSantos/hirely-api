package handlers

import (
	"context"
	"fmt"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"log/slog"

	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPHandler struct {
	appService *services.ApplicationService
	sseServer  *server.SSEServer
}

func NewMCPHandler(appService *services.ApplicationService) *MCPHandler {
	mcpServer := server.NewMCPServer("Hirely-Cloud-MCP", "1.0.0")

	readTool := mcp.NewTool("read_applications", mcp.WithDescription("Lê candidaturas de emprego."))
	mcpServer.AddTool(readTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, ok := ctx.Value("userIDKey").(string)
		if !ok || userID == "" {
			return mcp.NewToolResultError("Unauthorized: missing user ID in context"), nil
		}

		apps, err := appService.ListApplications(ctx, userID, "", nil, nil, "appliedAt", "desc")
		if err != nil {
			return mcp.NewToolResultError("Erro ao buscar vagas: " + err.Error()), nil
		}

		slog.Info("MCP Read Applications tool called", slog.String("traceId", logger.GetTraceID(ctx)))
		return mcp.NewToolResultText(fmt.Sprintf("Vagas encontradas: %d", len(apps))), nil
	})

	insertTool := mcp.NewTool("insert_application",
		mcp.WithDescription("Insere uma nova candidatura a uma vaga de emprego."),
		mcp.WithString("company", mcp.Required(), mcp.Description("Nome da empresa")),
		mcp.WithString("role", mcp.Required(), mcp.Description("Cargo ou título da vaga")),
		mcp.WithString("url", mcp.Required(), mcp.Description("Link da vaga")),
	)
	mcpServer.AddTool(insertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID, ok := ctx.Value("userIDKey").(string)
		if !ok || userID == "" {
			return mcp.NewToolResultError("Unauthorized: missing user ID in context"), nil
		}

		company, err := request.RequireString("company")
		if err != nil {
			return mcp.NewToolResultError("Missing company"), nil
		}
		role, err := request.RequireString("role")
		if err != nil {
			return mcp.NewToolResultError("Missing role"), nil
		}
		url, err := request.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError("Missing url"), nil
		}

		input := services.CreateApplicationInput{
			CompanyName: company,
			JobTitle:    role,
			JobURL:      url,
			Status:      domain.StatusToApply, // Assumindo ToApply por padrão
		}

		_, err = appService.CreateApplication(ctx, userID, input)
		if err != nil {
			return mcp.NewToolResultError("Falha ao inserir candidatura: " + err.Error()), nil
		}

		slog.Info("MCP Insert Application tool called", slog.String("traceId", logger.GetTraceID(ctx)))
		return mcp.NewToolResultText(fmt.Sprintf("Vaga para %s na %s criada com sucesso no Kanban!", role, company)), nil
	})

	dynamicBasePath := func(r *http.Request, sessionID string) string {
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		return scheme + "://" + r.Host
	}

	sseServer := server.NewSSEServer(mcpServer,
		server.WithSSEEndpoint("/v1/mcp/sse"),
		server.WithMessageEndpoint("/v1/mcp/messages"),
		server.WithDynamicBasePath(dynamicBasePath),
	)

	return &MCPHandler{
		appService: appService,
		sseServer:  sseServer,
	}
}

func (h *MCPHandler) HandleSSE() gin.HandlerFunc {
	return gin.WrapH(h.sseServer.SSEHandler())
}

func (h *MCPHandler) HandleMessage() gin.HandlerFunc {
	return gin.WrapH(h.sseServer.MessageHandler())
}
