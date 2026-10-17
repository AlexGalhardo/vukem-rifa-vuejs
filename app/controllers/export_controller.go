package controllers

import (
	"bytes"
	"net/http"

	"github.com/AlexGalhardo/vukem/auth"
	vexport "github.com/AlexGalhardo/vukem/export"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/export"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/models"
	"github.com/AlexGalhardo/vukem-rifa-vuejs/app/raffles"
)

// ExportController serves the owner's audit export downloads.
type ExportController struct {
	raffles *raffles.Service
	export  *export.Service
}

// NewExportController returns an ExportController.
func NewExportController(r *raffles.Service, e *export.Service) *ExportController {
	return &ExportController{raffles: r, export: e}
}

// JSON downloads the export as JSON.
func (ec *ExportController) JSON(c *web.Ctx) error { return ec.download(c, "json") }

// CSV downloads the export as CSV.
func (ec *ExportController) CSV(c *web.Ctx) error { return ec.download(c, "csv") }

func (ec *ExportController) download(c *web.Ctx, format string) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	id, err := c.ParamInt("raffle")
	if err != nil {
		return err
	}

	r, err := ec.raffles.ForOwner(c.Context(), u.ID, id)
	if err != nil {
		return err
	}

	e, err := ec.export.Build(c.Context(), r)
	if err != nil {
		return err
	}

	return Download(c, e, format)
}

// Download writes e as an attachment in format ("json" or "csv"). Exports
// hold buyers' personal data: vukem export sends them with no-store,
// nosniff and an RFC 5987 Content-Disposition, and guards every CSV cell.
func Download(c *web.Ctx, e export.Export, format string) error {
	name := export.Filename(e, format)

	if format == "csv" {
		return vexport.WriteCSV(c.Writer(), c.Request(), name, func(cw *vexport.CSVWriter) error {
			return export.Records(e, cw.Write)
		})
	}

	var buf bytes.Buffer
	if err := export.WriteJSON(&buf, e); err != nil {
		return err
	}

	h := c.Writer().Header()
	h.Set("Content-Disposition", vexport.ContentDisposition(name))
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")

	return c.Blob(http.StatusOK, "application/json; charset=utf-8", buf.Bytes())
}
