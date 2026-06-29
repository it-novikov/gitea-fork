// Copyright 2026 The KYBa Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package kcp

import (
	"net/http"

	"code.gitea.io/gitea/modules/kcp/webui"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/context"
)

// RegisterRoutes registers KYBa KCP API endpoints under /api/v1/kcp.
func RegisterRoutes(m *web.Router) {
	m.Get("/capsules", ListCapsules)
	m.Get("/imports", ListImports)
	m.Get("/impact", Impact)
	m.Get("/export-plan", ExportPlan)
}

func ListCapsules(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, webui.BuildViewModel(webui.PageCapsules, webui.DemoDataSet()).Capsules)
}

func ListImports(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, webui.BuildViewModel(webui.PageImports, webui.DemoDataSet()).Imports)
}

func Impact(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, webui.BuildViewModel(webui.PageImpact, webui.DemoDataSet()).ImpactRows)
}

func ExportPlan(ctx *context.APIContext) {
	model := webui.BuildViewModel(webui.PageExport, webui.DemoDataSet())
	ctx.JSON(http.StatusOK, map[string]any{
		"ready":            model.ExportReady,
		"ownership_digest": model.ExportDigest,
		"targets":          model.ExportRows,
	})
}
