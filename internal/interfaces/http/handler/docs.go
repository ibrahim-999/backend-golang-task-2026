package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type Docs struct {
	specPath string
}

func NewDocs(specPath string) *Docs {
	return &Docs{specPath: specPath}
}

func (d *Docs) Spec(c *gin.Context) {
	data, err := os.ReadFile(d.specPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "openapi specification not found"})
		return
	}
	c.Data(http.StatusOK, "application/yaml", data)
}

func (d *Docs) UI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIPage))
}

const swaggerUIPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Order Processing API Docs</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
<script src="https://unpkg.com/swagger-ui-dist/swagger-ui-standalone-preset.js"></script>
<script>
window.onload = function () {
  window.ui = SwaggerUIBundle({
    url: "/openapi.yaml",
    dom_id: "#swagger-ui",
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    layout: "StandaloneLayout"
  });
};
</script>
</body>
</html>`
