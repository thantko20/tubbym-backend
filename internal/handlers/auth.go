package handlers

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/thantko20/tubbym-backend/internal/domain"
)

func (h *Handlers) LoginWithProvider(c *fiber.Ctx) error {
	provider := c.Params("provider")

	url, err := h.authService.LoginWithProvider(domain.AuthProvider(provider))
	if err != nil {
		slog.Error("Failed to get login URL", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal Server Error",
			"code":    9999,
		})
	}

	return c.Redirect(url, fiber.StatusFound)
}

func (h *Handlers) HandleProviderCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	code := c.Query("code")
	// state := c.Query("state")

	// if state != "state" {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"success": false,
	// 		"message": "Invalid state parameter",
	// 		"code":    domain.ErrCodeValidation,
	// 	})
	// }

	session, err := h.authService.HandleProviderCallback(c.Context(), domain.AuthProvider(provider), code)
	if err != nil {
		slog.Error("Failed to get user info", "error", err)
		return c.Redirect("http://localhost:3000/login?error=failed_to_get_user_info", fiber.StatusTemporaryRedirect)
	}

	cookie := new(fiber.Cookie)
	cookie.Name = "t_session_id"
	cookie.Value = session.Token
	cookie.Expires = session.ExpiredAt
	cookie.HTTPOnly = true
	cookie.SameSite = "Lax"

	c.Cookie(cookie)

	return c.Redirect("http://localhost:3000/", fiber.StatusFound)
}

func (h *Handlers) Logout(c *fiber.Ctx) error {
	err := h.authService.Logout(c.Context(), c.Cookies("t_session_id"))

	if err != nil {
		slog.Error("Failed to logout", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal Server Error",
			"code":    9999,
		})
	}

	c.ClearCookie("t_session_id")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Logged out successfully",
	})
}
