package handlers

import (
	"context"
	"fmt"
	"hirely-api/internal/adapters/http/middleware"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPHandler struct {
	appService *services.ApplicationService
	sseServer  *server.SSEServer
}

type mcpApplicationView struct {
	ID           string     `json:"id"`
	CompanyName  string     `json:"companyName"`
	JobTitle     string     `json:"jobTitle"`
	JobURL       string     `json:"jobUrl,omitempty"`
	Status       string     `json:"status"`
	SalaryRange  string     `json:"salaryRange,omitempty"`
	ContractType string     `json:"contractType,omitempty"`
	Location     string     `json:"location,omitempty"`
	AppliedAt    *time.Time `json:"appliedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func toMCPApplicationView(app *domain.Application) mcpApplicationView {
	view := mcpApplicationView{
		ID:          app.ID,
		CompanyName: app.CompanyName,
		JobTitle:    app.JobTitle,
		JobURL:      app.JobURL,
		Status:      string(app.Status),
		SalaryRange: app.SalaryRange,
		Location:    app.Location,
		AppliedAt:   app.AppliedAt,
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
	}
	if app.ContractType != nil {
		view.ContractType = string(*app.ContractType)
	}
	return view
}

func isValidStatus(status domain.ApplicationStatus) bool {
	for _, s := range domain.AllStatuses() {
		if s == status {
			return true
		}
	}
	return false
}

func applicationsListText(apps []mcpApplicationView) string {
	if len(apps) == 0 {
		return "Nenhuma candidatura encontrada."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d candidatura(s) encontrada(s):\n", len(apps))
	for i, app := range apps {
		appliedAt := ""
		if app.AppliedAt != nil {
			appliedAt = app.AppliedAt.Format("2006-01-02")
		}
		fmt.Fprintf(&sb, "%d. %s - %s (%s)%s\n", i+1, app.CompanyName, app.JobTitle, app.Status, appliedAt)
		if app.JobURL != "" {
			fmt.Fprintf(&sb, "   URL: %s\n", app.JobURL)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func NewMCPHandler(appService *services.ApplicationService, backEndURL string) *MCPHandler {
	mcpServer := server.NewMCPServer("Hirely-Cloud-MCP", "1.0.0")

	readTool := mcp.NewTool("read_applications",
		mcp.WithDescription("Lista as candidaturas de emprego do usuário com empresa, cargo, link, status, salário, local e datas."),
		mcp.WithString("search", mcp.Description("Filtra candidaturas por texto em empresa ou cargo")),
		mcp.WithString("status", mcp.Enum(
			string(domain.StatusToApply),
			string(domain.StatusApplied),
			string(domain.StatusInterview),
			string(domain.StatusOffer),
			string(domain.StatusAccepted),
			string(domain.StatusRejected),
			string(domain.StatusOther),
		), mcp.Description("Filtra candidaturas por status")),
	)
	mcpServer.AddTool(readTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID := middleware.GetUserID(ctx)
		if userID == "" {
			return mcp.NewToolResultError("Unauthorized: missing user ID in context"), nil
		}

		search := request.GetString("search", "")
		var statusFilters []string
		if st := strings.TrimSpace(request.GetString("status", "")); st != "" {
			statusFilters = []string{st}
		}

		apps, err := appService.ListApplications(ctx, userID, search, statusFilters, nil, "appliedAt", "desc")
		if err != nil {
			return mcp.NewToolResultError("Erro ao buscar candidaturas: " + err.Error()), nil
		}

		views := make([]mcpApplicationView, 0, len(apps))
		for _, app := range apps {
			views = append(views, toMCPApplicationView(app))
		}

		slog.Info("MCP Read Applications tool called", slog.String("traceId", logger.GetTraceID(ctx)))
		return mcp.NewToolResultStructured(
			map[string]any{"total": len(views), "applications": views},
			applicationsListText(views),
		), nil
	})

	insertTool := mcp.NewTool("insert_application",
		mcp.WithDescription("Cria uma nova candidatura a uma vaga de emprego."),
		mcp.WithString("company", mcp.Required(), mcp.Description("Nome da empresa")),
		mcp.WithString("role", mcp.Required(), mcp.Description("Cargo ou título da vaga")),
		mcp.WithString("url", mcp.Required(), mcp.Description("Link da vaga")),
		mcp.WithString("status", mcp.Enum(
			string(domain.StatusToApply),
			string(domain.StatusApplied),
			string(domain.StatusInterview),
			string(domain.StatusOffer),
			string(domain.StatusAccepted),
			string(domain.StatusRejected),
			string(domain.StatusOther),
		), mcp.Description("Status da candidatura (padrão: TO_APPLY)")),
		mcp.WithString("salaryRange", mcp.Description("Faixa salarial, ex: R$ 8.000 - R$ 10.000")),
		mcp.WithString("contractType", mcp.Enum(
			string(domain.ContractTypeCLT),
			string(domain.ContractTypePJ),
			string(domain.ContractTypeInternship),
			string(domain.ContractTypeOther),
		), mcp.Description("Tipo de contrato: CLT, PJ, INTERNSHIP ou OTHER")),
		mcp.WithString("location", mcp.Description("Local da vaga (cidade, remoto, híbrido, etc.)")),
		mcp.WithString("appliedAt", mcp.Description("Data em que se candidatou em formato RFC3339, ex: 2026-09-05T10:00:00Z")),
		mcp.WithString("notes", mcp.Description("Anotações livres sobre a vaga")),
	)
	mcpServer.AddTool(insertTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID := middleware.GetUserID(ctx)
		if userID == "" {
			return mcp.NewToolResultError("Unauthorized: missing user ID in context"), nil
		}

		company, err := request.RequireString("company")
		if err != nil {
			return mcp.NewToolResultError("Missing required field: company"), nil
		}
		role, err := request.RequireString("role")
		if err != nil {
			return mcp.NewToolResultError("Missing required field: role"), nil
		}
		url, err := request.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError("Missing required field: url"), nil
		}

		status := domain.ApplicationStatus(request.GetString("status", string(domain.StatusToApply)))
		if !isValidStatus(status) {
			return mcp.NewToolResultError("Invalid status. Use one of: TO_APPLY, APPLIED, INTERVIEW, OFFER, ACCEPTED, REJECTED, OTHER"), nil
		}

		var contractType *domain.ContractType
		if ct := strings.TrimSpace(request.GetString("contractType", "")); ct != "" {
			ctValue := domain.ContractType(ct)
			if !ctValue.IsValid() {
				return mcp.NewToolResultError("Invalid contractType. Use one of: CLT, PJ, INTERNSHIP, OTHER"), nil
			}
			contractType = &ctValue
		}

		var appliedAt *time.Time
		if raw := strings.TrimSpace(request.GetString("appliedAt", "")); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return mcp.NewToolResultError("Invalid appliedAt. Use RFC3339 format, ex: 2026-09-05T10:00:00Z"), nil
			}
			appliedAt = &t
		}

		created, err := appService.CreateApplication(ctx, userID, services.CreateApplicationInput{
			CompanyName:  company,
			JobTitle:     role,
			JobURL:       url,
			Status:       status,
			ContractType: contractType,
			SalaryRange:  strings.TrimSpace(request.GetString("salaryRange", "")),
			Location:     strings.TrimSpace(request.GetString("location", "")),
			Notes:        request.GetString("notes", ""),
			AppliedAt:    appliedAt,
		})
		if err != nil {
			return mcp.NewToolResultError("Falha ao inserir candidatura: " + err.Error()), nil
		}

		view := toMCPApplicationView(created)
		slog.Info("MCP Insert Application tool called", slog.String("traceId", logger.GetTraceID(ctx)))
		return mcp.NewToolResultStructured(
			map[string]any{"application": view},
			fmt.Sprintf("Candidatura criada: %s - %s (id: %s, status: %s)", created.CompanyName, created.JobTitle, created.ID, created.Status),
		), nil
	})

	sseServer := server.NewSSEServer(mcpServer,
		server.WithSSEEndpoint("/v1/mcp/sse"),
		server.WithMessageEndpoint("/v1/mcp/messages"),
		server.WithBaseURL(backEndURL),
	)

	return &MCPHandler{
		appService: appService,
		sseServer:  sseServer,
	}
}

func (h *MCPHandler) HandleSSE() gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Info("MCP SSE connection attempt",
			slog.String("path", c.Request.URL.Path),
			slog.String("query", c.Request.URL.RawQuery),
		)
		gin.WrapH(h.sseServer.SSEHandler())(c)
	}
}

func (h *MCPHandler) HandleMessage() gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Info("MCP Message received",
			slog.String("path", c.Request.URL.Path),
			slog.String("query", c.Request.URL.RawQuery),
			slog.String("sessionId", c.Query("sessionId")),
		)
		gin.WrapH(h.sseServer.MessageHandler())(c)
	}
}
