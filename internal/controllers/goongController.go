package controllers

import (
	"history-api/internal/services"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type GoongController interface {
	Proxy(c fiber.Ctx) error
}

type goongController struct {
	goongService services.GoongService
}

func NewGoongController(goongService services.GoongService) GoongController {
	return &goongController{
		goongService: goongService,
	}
}

// Proxy godoc
// @Summary Proxy Goong APIs
// @Description Transparent proxy for Goong APIs to forward body, params, headers and inject API key automatically.
// @Tags Goong
// @Accept json
// @Produce json
// @Produce application/x-protobuf
// @Param path path string true "Target URL to proxy, e.g., 'tiles.goong.io/assets/goong_map_web.json'"
// @Success 200 {string} string "Resource content"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "Forbidden"
// @Failure 500 {string} string "Internal Server Error"
// @Router /proxy/{path} [get]
func (ctrl *goongController) Proxy(c fiber.Ctx) error {
	path := c.Params("*")
	if path == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid proxy URL format")
	}

	targetURL := path
	if decodedURL, err := url.PathUnescape(targetURL); err == nil {
		targetURL = decodedURL
	}

	if strings.HasPrefix(targetURL, "https:/") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = strings.Replace(targetURL, "https:/", "https://", 1)
	} else if strings.HasPrefix(targetURL, "http:/") && !strings.HasPrefix(targetURL, "http://") {
		targetURL = strings.Replace(targetURL, "http:/", "http://", 1)
	}

	if len(c.Request().URI().QueryString()) > 0 {
		if strings.Contains(targetURL, "?") {
			targetURL += "&" + string(c.Request().URI().QueryString())
		} else {
			targetURL += "?" + string(c.Request().URI().QueryString())
		}
	}

	headers := make(map[string]string)
	for k, v := range c.GetReqHeaders() {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	statusCode, respHeaders, respBody, err := ctrl.goongService.ProxyRequest(
		c.Context(),
		c.Method(),
		targetURL,
		headers,
		c.Body(),
	)

	if err != nil {
		return c.Status(statusCode).SendString(err.Error())
	}

	for k, v := range respHeaders {
		c.Set(k, v)
	}

	c.Set("Vary", "Origin")
	c.Set("Cross-Origin-Resource-Policy", "cross-origin")

	if c.Method() == "GET" && statusCode == fiber.StatusOK {
		isStaticAsset := strings.Contains(targetURL, "/tiles/") ||
			strings.Contains(targetURL, "/fonts/") ||
			strings.Contains(targetURL, "/glyphs/") ||
			strings.HasSuffix(targetURL, ".pbf") ||
			strings.HasSuffix(targetURL, ".png") ||
			strings.HasSuffix(targetURL, ".jpg") ||
			strings.HasSuffix(targetURL, ".webp")

		if !isStaticAsset {
			c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			c.Set("CDN-Cache-Control", "no-store")
			c.Set("Cloudflare-CDN-Cache-Control", "no-store")
		} else {
			c.Set("Cache-Control", "public, max-age=86400")
			c.Set("CDN-Cache-Control", "max-age=86400")
			c.Set("Cloudflare-CDN-Cache-Control", "max-age=86400")
		}
	} else if c.Method() == "GET" {
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		c.Set("CDN-Cache-Control", "no-store")
		c.Set("Cloudflare-CDN-Cache-Control", "no-store")
	}

	return c.Status(statusCode).Send(respBody)
}
