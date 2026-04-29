package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
)

type RuntimeConfig = libauth.RuntimeConfig

func Initialize(ctx context.Context, cfg RuntimeConfig) error {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		generated, err := libauth.RandomHex(32)
		if err != nil {
			return err
		}
		secret = generated
	}
	middleware.SetCookieSecret([]byte(secret))

	if _, _, _, err := libauth.EnsureBootstrapState(ctx); err != nil {
		return err
	}
	libauth.RunAuditRetentionCleanup(ctx, cfg)
	return nil
}

func Routes(route *gin.RouterGroup) {
	authRoutes := route.Group("/auth")
	authRoutes.GET("/bootstrap-status", getBootstrapStatus)
	authRoutes.GET("/bootstrap-options", getBootstrapOptions)
	authRoutes.POST("/bootstrap/select-user", bootstrapSelectUser)
	authRoutes.POST("/bootstrap/create-user", bootstrapCreateUser)
	authRoutes.GET("/session", getSession)
	authRoutes.POST("/logout", logout)

	authRoutes.POST("/register/begin", beginRegistration)
	authRoutes.POST("/register/finish", finishRegistration)
	authRoutes.POST("/login/begin", beginLogin)
	authRoutes.POST("/login/finish", finishLogin)

	self := authRoutes.Group("/self")
	self.Use(middleware.RequireAuthenticated())
	self.POST("/assignment-link", createSelfAssignmentLink)
	self.GET("/assignment-link", getSelfAssignmentLink)
	self.DELETE("/assignment-link", middleware.RequireConfirmedAction(), deleteSelfAssignmentLink)
	self.GET("/passkeys", listSelfPasskeys)
	self.POST("/passkeys/:credential_id/revoke", middleware.RequireConfirmedAction(), revokeSelfPasskey)

	admin := authRoutes.Group("/admin")
	admin.Use(middleware.RequireAuthenticated(), middleware.RequireRole(libauth.RoleAdmin))
	admin.GET("/users/:user_id/assignment-link", adminGetActiveAssignmentLink)
	admin.POST("/users/:user_id/assignment-link", adminCreateAssignmentLink)
	admin.DELETE("/users/:user_id/assignment-link", middleware.RequireConfirmedAction(), adminDeleteActiveAssignmentLink)
	admin.GET("/users/:user_id/passkeys", adminListUserPasskeys)
	admin.POST("/assignment-links/:link_id/revoke", middleware.RequireConfirmedAction(), adminRevokeAssignmentLink)
	admin.POST("/users/:user_id/passkeys/revoke-all", middleware.RequireConfirmedAction(), adminRevokeAllPasskeys)
	admin.POST("/passkeys/:credential_id/revoke", middleware.RequireConfirmedAction(), adminRevokePasskey)
	admin.GET("/audit-logs", adminListAuditLogs)
}

func getBootstrapStatus(c *gin.Context) {
	required, token, userID, err := libauth.EnsureBootstrapState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bootstrap_required": required,
		"assignment_token":   token,
		"user_id":            userID,
		"assignment_url":     "/app?assignment_token=" + token,
	})
}

func getBootstrapOptions(c *gin.Context) {
	required, token, userID, err := libauth.EnsureBootstrapState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	users, err := libauth.ListBootstrapUserOptions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"bootstrap_required": required,
		"assignment_token":   token,
		"user_id":            userID,
		"users":              users,
	})
}

func bootstrapSelectUser(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	required, _, _, err := libauth.EnsureBootstrapState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !required {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bootstrap flow is not active"})
		return
	}

	uid, msg, ok := id_helper.MustParseAndMarshalUUID(req.UserID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	token, selectedUserID, err := libauth.SelectBootstrapUser(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"assignment_token": token, "user_id": selectedUserID})
}

func bootstrapCreateUser(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)
	required, _, _, err := libauth.EnsureBootstrapState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !required {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bootstrap flow is not active"})
		return
	}
	token, selectedUserID, err := libauth.CreateBootstrapUserAndSelect(c, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"assignment_token": token, "user_id": selectedUserID})
}

func beginRegistration(c *gin.Context) {
	var req struct {
		AssignmentToken string `json:"assignment_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := libauth.BeginRegistration(c, req.AssignmentToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"challenge_token": res.ChallengeToken,
		"challenge":       res.Challenge,
		"user_id":         res.UserID,
		"user_name":       res.UserName,
	})
}

func finishRegistration(c *gin.Context) {
	var req struct {
		ChallengeToken string   `json:"challenge_token"`
		Challenge      string   `json:"challenge"`
		CredentialID   string   `json:"credential_id"`
		PublicKey      string   `json:"public_key"`
		Label          string   `json:"label"`
		Transports     []string `json:"transports"`
		ClientDataJSON string   `json:"client_data_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	credentialID, err := libauth.DecodeBase64(req.CredentialID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential_id"})
		return
	}
	userID, err := libauth.FinishRegistration(c, libauth.FinishRegistrationInput{
		ChallengeToken: req.ChallengeToken,
		Challenge:      req.Challenge,
		CredentialID:   req.CredentialID,
		PublicKey:      req.PublicKey,
		Label:          req.Label,
		Transports:     req.Transports,
		ClientDataJSON: req.ClientDataJSON,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := middleware.SetSessionCookieWithCredential(c, userID, credentialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func beginLogin(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := libauth.BeginLogin(c, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"challenge_token": res.ChallengeToken, "challenge": res.Challenge})
}

func finishLogin(c *gin.Context) {
	var req struct {
		ChallengeToken    string `json:"challenge_token"`
		Challenge         string `json:"challenge"`
		CredentialID      string `json:"credential_id"`
		SignCounter       int64  `json:"sign_counter"`
		ClientDataJSON    string `json:"client_data_json"`
		AuthenticatorData string `json:"authenticator_data"`
		Signature         string `json:"signature"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	credentialID, err := libauth.DecodeBase64(req.CredentialID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential_id"})
		return
	}
	userID, err := libauth.FinishLogin(c, libauth.FinishLoginInput{
		ChallengeToken:    req.ChallengeToken,
		Challenge:         req.Challenge,
		CredentialID:      req.CredentialID,
		SignCounter:       req.SignCounter,
		ClientDataJSON:    req.ClientDataJSON,
		AuthenticatorData: req.AuthenticatorData,
		Signature:         req.Signature,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := middleware.SetSessionCookieWithCredential(c, userID, credentialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func getSession(c *gin.Context) {
	if middleware.AuthBypassed() {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}

	userID, err := middleware.ParseSessionCookie(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}
	user, err := database.Connection.GetUserByID(c, userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}
	roles, _ := database.Connection.ListUserRoles(c, userID)
	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user":          gin.H{"ID": user.ID, "Name": user.Name},
		"roles":         roles,
	})
}

func logout(c *gin.Context) {
	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func createSelfAssignmentLink(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	row, rawToken, err := libauth.CreateStandardAssignmentLinkForUser(c, userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, assignmentLinkResponse(c, row, rawToken))
}

func getSelfAssignmentLink(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	row, err := database.Connection.GetActiveStandardAssignmentLinkByUserID(c, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusOK, gin.H{"has_active_link": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// The raw token is only available at creation time; we cannot recover it
	// from the stored hash. Return metadata without the token.
	res := assignmentLinkResponse(c, row, "")
	res["has_active_link"] = true
	c.JSON(http.StatusOK, res)
}

func deleteSelfAssignmentLink(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	revoked, err := database.Connection.RevokeActiveStandardAssignmentLinksForUser(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if revoked > 0 {
		_ = libauth.AuditLog(c, &userID, "assignment_link_revoked", "user", libauth.IDOrEmpty(userID), map[string]any{"revoked_count": revoked, "scope": "self"})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "revoked_count": revoked})
}

func listSelfPasskeys(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	rows, err := database.Connection.ListPasskeysByUser(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	activeCredentialID, _ := middleware.CurrentCredentialID(c)
	c.JSON(http.StatusOK, passkeyListResponse(rows, activeCredentialID))
}

func revokeSelfPasskey(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	credentialID, credentialIDHex, err := credentialIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential_id"})
		return
	}

	passkey, err := database.Connection.GetPasskeyByCredentialID(c, credentialID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "passkey not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !bytes.Equal(passkey.UserID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	activeCredentialID, ok := middleware.CurrentCredentialID(c)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "reauthentication required before removing passkeys"})
		return
	}
	if bytes.Equal(activeCredentialID, credentialID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove the passkey used by your current session"})
		return
	}

	if err := database.Connection.RevokePasskey(c, credentialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = libauth.AuditLog(c, &userID, "passkey_revoked", "passkey", credentialIDHex, map[string]any{"scope": "self"})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminCreateAssignmentLink(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	row, rawToken, err := libauth.CreateStandardAssignmentLinkForUser(c, uid, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, assignmentLinkResponse(c, row, rawToken))
}

func adminGetActiveAssignmentLink(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	row, err := database.Connection.GetActiveStandardAssignmentLinkByUserID(c, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusOK, gin.H{"has_active_link": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The raw token is only available at creation time; return metadata only.
	res := assignmentLinkResponse(c, row, "")
	res["has_active_link"] = true
	c.JSON(http.StatusOK, res)
}

func adminDeleteActiveAssignmentLink(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	revoked, err := database.Connection.RevokeActiveStandardAssignmentLinksForUser(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if revoked > 0 {
		actor, _ := middleware.CurrentUserID(c)
		_ = libauth.AuditLog(c, &actor, "assignment_link_revoked", "user", c.Param("user_id"), map[string]any{"revoked_count": revoked})
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "revoked_count": revoked})
}

func adminRevokeAssignmentLink(c *gin.Context) {
	linkID, err := strconv.ParseInt(c.Param("link_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link_id"})
		return
	}
	if err := database.Connection.RevokeAssignmentLink(c, linkID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "assignment_link_revoked", "assignment_link", c.Param("link_id"), map[string]any{})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminRevokeAllPasskeys(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if err := database.Connection.RevokeAllPasskeysForUser(c, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "passkeys_revoked_all", "user", c.Param("user_id"), map[string]any{})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminRevokePasskey(c *gin.Context) {
	credentialID, credentialIDHex, err := credentialIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential_id"})
		return
	}
	if err := database.Connection.RevokePasskey(c, credentialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "passkey_revoked", "passkey", credentialIDHex, map[string]any{"scope": "admin"})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminListUserPasskeys(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	rows, err := database.Connection.ListPasskeysByUser(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, passkeyListResponse(rows, nil))
}

func adminListAuditLogs(c *gin.Context) {
	limit := int32(100)
	offset := int32(0)
	if s := c.Query("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			limit = int32(v)
		}
	}
	if s := c.Query("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			offset = int32(v)
		}
	}

	rows, err := database.Connection.ListAuditLogs(c, database.ListAuditLogsParams{Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// assignmentLinkResponse builds the JSON response for an assignment link.
// rawToken should be the raw (unhashed) token returned at creation time; pass
// an empty string for read-only endpoints where the token cannot be recovered
// from the stored hash.
func assignmentLinkResponse(c *gin.Context, row database.PasskeyAssignmentLink, rawToken string) gin.H {
	res := gin.H{
		"link_id": row.ID,
	}
	if rawToken != "" {
		res["assignment_token"] = rawToken
		res["assignment_url"] = buildAssignmentURL(c, rawToken)
	}
	if row.ExpiresAt.Valid {
		res["expires_at"] = row.ExpiresAt.Time
	}
	return res
}

func buildAssignmentURL(c *gin.Context, token string) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host + "/app?assignment_token=" + url.QueryEscape(token)
}

func passkeyListResponse(rows []database.Passkey, activeCredentialID []byte) []gin.H {
	res := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		if row.RevokedAt.Valid {
			continue
		}
		label := ""
		if row.Label.Valid {
			label = row.Label.String
		}
		res = append(res, gin.H{
			"credential_id":      hex.EncodeToString(row.CredentialID),
			"label":              label,
			"created_at":         row.CreatedAt,
			"is_current_session": len(activeCredentialID) > 0 && bytes.Equal(row.CredentialID, activeCredentialID),
		})
	}
	return res
}

func credentialIDFromParam(c *gin.Context) ([]byte, string, error) {
	credentialIDRaw := c.Param("credential_id")
	credentialID, err := hex.DecodeString(credentialIDRaw)
	if err == nil {
		return credentialID, credentialIDRaw, nil
	}
	credentialID, err = libauth.DecodeBase64(credentialIDRaw)
	if err != nil {
		return nil, "", err
	}
	return credentialID, hex.EncodeToString(credentialID), nil
}

