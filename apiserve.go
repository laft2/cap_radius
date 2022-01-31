package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func APIServer() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.POST("/api/post_context", func(c echo.Context) error {
		detailRow := c.FormValue("detail")
		authRow := c.FormValue("auth")

		UpdateContext(detailRow, authRow)
		c.NoContent(http.StatusOK)

		return nil
	})
	e.Logger.Fatal(e.StartTLS(":9090", "server.crt", "server.key"))
}
