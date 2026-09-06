package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type C2CHandler struct {
	service *service.C2CService
}

func NewC2CHandler(service *service.C2CService) *C2CHandler {
	return &C2CHandler{service: service}
}

func (h *C2CHandler) ListProviderAccounts(c *gin.Context) {
	userID, ok := c2cUserID(c)
	if !ok {
		return
	}
	items, err := h.service.ListProviderAccounts(c.Request.Context(), userID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, items)
}

func (h *C2CHandler) CreateProviderAccount(c *gin.Context) {
	userID, ok := c2cUserID(c)
	if !ok {
		return
	}
	var input service.C2CCreateProviderAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	item, err := h.service.CreateProviderAccount(c.Request.Context(), userID, input)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *C2CHandler) VerifyProviderAccount(c *gin.Context) {
	userID, accountID, ok := c2cUserAndAccountID(c)
	if !ok {
		return
	}
	var input struct {
		Model string `json:"model"`
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			response.BadRequest(c, "invalid request: "+err.Error())
			return
		}
	}
	item, err := h.service.VerifyProviderAccount(c.Request.Context(), userID, accountID, strings.TrimSpace(input.Model))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *C2CHandler) SetPoolState(c *gin.Context) {
	userID, accountID, ok := c2cUserAndAccountID(c)
	if !ok {
		return
	}
	var input struct {
		Active *bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Active == nil {
		response.BadRequest(c, "active is required")
		return
	}
	item, err := h.service.SetPoolState(c.Request.Context(), userID, accountID, *input.Active)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *C2CHandler) DeleteProviderAccount(c *gin.Context) {
	userID, accountID, ok := c2cUserAndAccountID(c)
	if !ok {
		return
	}
	if response.ErrorFrom(c, h.service.DeleteProviderAccount(c.Request.Context(), userID, accountID)) {
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *C2CHandler) Wallet(c *gin.Context) {
	userID, ok := c2cUserID(c)
	if !ok {
		return
	}
	wallet, err := h.service.Wallet(c.Request.Context(), userID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, wallet)
}

func (h *C2CHandler) Stats(c *gin.Context) {
	userID, ok := c2cUserID(c)
	if !ok {
		return
	}
	stats, err := h.service.Stats(c.Request.Context(), userID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, stats)
}

func (h *C2CHandler) Ledger(c *gin.Context) {
	userID, ok := c2cUserID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.service.Ledger(c.Request.Context(), userID, limit)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, items)
}

func c2cUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "user not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func c2cUserAndAccountID(c *gin.Context) (int64, int64, bool) {
	userID, ok := c2cUserID(c)
	if !ok {
		return 0, 0, false
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "invalid account id")
		return 0, 0, false
	}
	return userID, accountID, true
}
