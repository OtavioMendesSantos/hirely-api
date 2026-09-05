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
	tagService *services.TagService
	sseServer  *server.SSEServer
}

type mcpTagView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ColorHex string `json:"colorHex"`
}

type mcpApplicationView struct {
	ID           string       `json:"id"`
	CompanyName  string       `json:"companyName"`
	JobTitle     string       `json:"jobTitle"`
	JobURL       string       `json:"jobUrl,omitempty"`
	Status       string       `json:"status"`
	SalaryRange  string       `json:"salaryRange,omitempty"`
	ContractType string       `json:"contractType,omitempty"`
	Location     string       `json:"location,omitempty"`
	Description  string       `json:"description,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	AppliedAt    *time.Time   `json:"appliedAt,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	Tags         []mcpTagView `json:"tags,omitempty"`
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
		Description: app.JobDescription,
		Notes:       app.Notes,
		AppliedAt:   app.AppliedAt,
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
	}
	if app.ContractType != nil {
		view.ContractType = string(*app.ContractType)
	}
	if len(app.Tags) > 0 {
		view.Tags = make([]mcpTagView, len(app.Tags))
		for i, t := range app.Tags {
			view.Tags[i] = mcpTagView{ID: t.ID, Name: t.Name, ColorHex: t.ColorHex}
		}
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
		tagStr := ""
		if len(app.Tags) > 0 {
			var tNames []string
			for _, t := range app.Tags {
				tNames = append(tNames, t.Name)
			}
			tagStr = " [" + strings.Join(tNames, ", ") + "]"
		}
		fmt.Fprintf(&sb, "%d. %s - %s (%s)%s%s\n", i+1, app.CompanyName, app.JobTitle, app.Status, appliedAt, tagStr)
		if app.JobURL != "" {
			fmt.Fprintf(&sb, "   URL: %s\n", app.JobURL)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func NewMCPHandler(appService *services.ApplicationService, tagService *services.TagService, backEndURL string) *MCPHandler {
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
		mcp.WithString("url", mcp.Description("Link da vaga")),
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
		mcp.WithString("description", mcp.Description("Descrição completa da vaga")),
		mcp.WithString("appliedAt", mcp.Description("Data em que se candidatou em formato RFC3339, ex: 2026-09-05T10:00:00Z")),
		mcp.WithString("notes", mcp.Description("Anotações livres sobre a vaga")),
		mcp.WithString("tagIds", mcp.Description("IDs das tags (separadas por vírgula)")),
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
		url := request.GetString("url", "")

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

		var tagIDs []string
		if rawTags := strings.TrimSpace(request.GetString("tagIds", "")); rawTags != "" {
			parts := strings.Split(rawTags, ",")
			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					tagIDs = append(tagIDs, t)
				}
			}
		}

		created, err := appService.CreateApplication(ctx, userID, services.CreateApplicationInput{
			CompanyName:  company,
			JobTitle:     role,
			JobURL:       url,
			Status:       status,
			ContractType: contractType,
			SalaryRange:    strings.TrimSpace(request.GetString("salaryRange", "")),
			Location:       strings.TrimSpace(request.GetString("location", "")),
			JobDescription: request.GetString("description", ""),
			Notes:          request.GetString("notes", ""),
			AppliedAt:      appliedAt,
			TagIDs:       tagIDs,
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

	updateTool := mcp.NewTool("update_application",
		mcp.WithDescription("Atualiza uma candidatura existente."),
		mcp.WithString("appId", mcp.Required(), mcp.Description("ID da candidatura")),
		mcp.WithString("company", mcp.Description("Nome da empresa")),
		mcp.WithString("role", mcp.Description("Cargo ou título da vaga")),
		mcp.WithString("url", mcp.Description("Link da vaga")),
		mcp.WithString("status", mcp.Enum(
			string(domain.StatusToApply),
			string(domain.StatusApplied),
			string(domain.StatusInterview),
			string(domain.StatusOffer),
			string(domain.StatusAccepted),
			string(domain.StatusRejected),
			string(domain.StatusOther),
		), mcp.Description("Status da candidatura")),
		mcp.WithString("salaryRange", mcp.Description("Faixa salarial")),
		mcp.WithString("contractType", mcp.Enum(
			string(domain.ContractTypeCLT),
			string(domain.ContractTypePJ),
			string(domain.ContractTypeInternship),
			string(domain.ContractTypeOther),
		), mcp.Description("Tipo de contrato")),
		mcp.WithString("location", mcp.Description("Local da vaga")),
		mcp.WithString("description", mcp.Description("Descrição completa da vaga")),
		mcp.WithString("appliedAt", mcp.Description("Data de candidatura (RFC3339)")),
		mcp.WithString("notes", mcp.Description("Anotações livres sobre a vaga")),
		mcp.WithString("tagIds", mcp.Description("IDs das tags (separadas por vírgula)")),
	)
	mcpServer.AddTool(updateTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID := middleware.GetUserID(ctx)
		if userID == "" {
			return mcp.NewToolResultError("Unauthorized: missing user ID in context"), nil
		}

		appID, err := request.RequireString("appId")
		if err != nil {
			return mcp.NewToolResultError("Missing required field: appId"), nil
		}

		input := services.UpdateApplicationInput{}

		if val := request.GetString("company", ""); val != "" {
			input.CompanyName = &val
		}
		if val := request.GetString("role", ""); val != "" {
			input.JobTitle = &val
		}
		if val := request.GetString("url", ""); val != "" {
			input.JobURL = &val
		}
		if val := request.GetString("status", ""); val != "" {
			st := domain.ApplicationStatus(val)
			if !isValidStatus(st) {
				return mcp.NewToolResultError("Invalid status"), nil
			}
			input.Status = &st
		}
		if val := request.GetString("salaryRange", ""); val != "" {
			input.SalaryRange = &val
		}
		if ct := strings.TrimSpace(request.GetString("contractType", "")); ct != "" {
			ctValue := domain.ContractType(ct)
			if !ctValue.IsValid() {
				return mcp.NewToolResultError("Invalid contractType"), nil
			}
			input.ContractType = &ctValue
		}
		if val := request.GetString("location", ""); val != "" {
			input.Location = &val
		}
		if val := request.GetString("description", ""); val != "" {
			input.JobDescription = &val
		}
		if val := request.GetString("notes", ""); val != "" {
			input.Notes = &val
		}
		if raw := strings.TrimSpace(request.GetString("appliedAt", "")); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return mcp.NewToolResultError("Invalid appliedAt format (RFC3339 required)"), nil
			}
			input.AppliedAt = &t
		}

		// Always check if tagIds is provided. If it's an empty string and was explicitly sent, it might clear tags?
		// Actually, let's just parse it if it's there. MCP tools don't have a reliable way to check if an optional string was provided empty vs not provided.
		// For simplicity, if it's not empty, we apply it. If it's literally "clear", we clear. Let's do: if it's provided, set it.
		// For our case, if we get rawTags, we just split.
		if rawTagsObj, ok := request.GetArguments()["tagIds"]; ok {
			if rawTags, ok := rawTagsObj.(string); ok {
			var tagIDs []string
			if strings.TrimSpace(rawTags) != "" {
				parts := strings.Split(rawTags, ",")
				for _, p := range parts {
					if t := strings.TrimSpace(p); t != "" {
						tagIDs = append(tagIDs, t)
					}
				}
			}
			input.TagIDs = tagIDs
			} else {
			    input.TagIDs = []string{}
			}
		}

		updated, err := appService.UpdateApplication(ctx, userID, appID, input)
		if err != nil {
			return mcp.NewToolResultError("Falha ao atualizar candidatura: " + err.Error()), nil
		}

		view := toMCPApplicationView(updated)
		slog.Info("MCP Update Application tool called", slog.String("traceId", logger.GetTraceID(ctx)))
		return mcp.NewToolResultStructured(
			map[string]any{"application": view},
			fmt.Sprintf("Candidatura atualizada: %s - %s (id: %s, status: %s)", updated.CompanyName, updated.JobTitle, updated.ID, updated.Status),
		), nil
	})

	listTagsTool := mcp.NewTool("list_tags",
		mcp.WithDescription("Lista as tags disponíveis do usuário para vincular a vagas."),
	)
	mcpServer.AddTool(listTagsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID := middleware.GetUserID(ctx)
		if userID == "" {
			return mcp.NewToolResultError("Unauthorized"), nil
		}

		tags, err := tagService.ListTags(ctx, userID)
		if err != nil {
			return mcp.NewToolResultError("Erro ao buscar tags: " + err.Error()), nil
		}

		var views []mcpTagView
		var sb strings.Builder
		if len(tags) == 0 {
			sb.WriteString("Nenhuma tag encontrada.")
		} else {
			fmt.Fprintf(&sb, "%d tag(s) encontrada(s):\n", len(tags))
			for i, t := range tags {
				views = append(views, mcpTagView{ID: t.ID, Name: t.Name, ColorHex: t.ColorHex})
				fmt.Fprintf(&sb, "%d. %s (ID: %s, Cor: %s)\n", i+1, t.Name, t.ID, t.ColorHex)
			}
		}

		slog.Info("MCP List Tags tool called", slog.String("traceId", logger.GetTraceID(ctx)))
		return mcp.NewToolResultStructured(
			map[string]any{"total": len(views), "tags": views},
			strings.TrimRight(sb.String(), "\n"),
		), nil
	})

	createTagTool := mcp.NewTool("create_tag",
		mcp.WithDescription("Cria uma nova tag para classificar vagas."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Nome da tag")),
		mcp.WithString("colorHex", mcp.Required(), mcp.Description("Cor da tag em HEX, ex: #FF0000")),
	)
	mcpServer.AddTool(createTagTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID := middleware.GetUserID(ctx)
		if userID == "" {
			return mcp.NewToolResultError("Unauthorized"), nil
		}

		name, err := request.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError("Missing name"), nil
		}
		colorHex, err := request.RequireString("colorHex")
		if err != nil {
			return mcp.NewToolResultError("Missing colorHex"), nil
		}

		tag, err := tagService.CreateTag(ctx, userID, name, colorHex)
		if err != nil {
			return mcp.NewToolResultError("Erro ao criar tag: " + err.Error()), nil
		}

		slog.Info("MCP Create Tag tool called", slog.String("traceId", logger.GetTraceID(ctx)))
		return mcp.NewToolResultStructured(
			map[string]any{"tag": mcpTagView{ID: tag.ID, Name: tag.Name, ColorHex: tag.ColorHex}},
			fmt.Sprintf("Tag criada: %s (ID: %s)", tag.Name, tag.ID),
		), nil
	})

	deleteTagTool := mcp.NewTool("delete_tag",
		mcp.WithDescription("Deleta uma tag existente pelo seu ID."),
		mcp.WithString("tagId", mcp.Required(), mcp.Description("ID da tag a ser deletada")),
	)
	mcpServer.AddTool(deleteTagTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		userID := middleware.GetUserID(ctx)
		if userID == "" {
			return mcp.NewToolResultError("Unauthorized"), nil
		}

		tagID, err := request.RequireString("tagId")
		if err != nil {
			return mcp.NewToolResultError("Missing tagId"), nil
		}

		err = tagService.DeleteTag(ctx, userID, tagID)
		if err != nil {
			return mcp.NewToolResultError("Erro ao deletar tag: " + err.Error()), nil
		}

		slog.Info("MCP Delete Tag tool called", slog.String("traceId", logger.GetTraceID(ctx)))
		return mcp.NewToolResultText("Tag deletada com sucesso."), nil
	})

	sseServer := server.NewSSEServer(mcpServer,
		server.WithSSEEndpoint("/v1/mcp/sse"),
		server.WithMessageEndpoint("/v1/mcp/messages"),
		server.WithBaseURL(backEndURL),
	)

	return &MCPHandler{
		appService: appService,
		tagService: tagService,
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
