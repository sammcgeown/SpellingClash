package handlers

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"

	"spellingclash/internal/utils"
)

// OAuthProvider defines provider configuration and metadata
type OAuthProvider struct {
	Name        string
	Label       string
	Config      *oauth2.Config
	UserInfoURL string
	AuthParams  map[string]string
}

type OAuthProviderView struct {
	Name     string
	Label    string
	URL      string
	CSSClass string
}

type oauthUserInfo struct {
	Subject string
	Email   string
	Name    string
}

func (h *AuthHandler) oauthProviderViews(r *http.Request) []OAuthProviderView {
	var views []OAuthProviderView
	familyCode := r.URL.Query().Get("family_code")

	for key, provider := range h.oauthProviders {
		if provider.Config == nil || provider.Config.ClientID == "" || provider.Config.ClientSecret == "" {
			continue
		}
		startURL := fmt.Sprintf("/auth/%s/start", key)
		if familyCode != "" {
			startURL = startURL + "?" + url.Values{"family_code": []string{familyCode}}.Encode()
		}
		views = append(views, OAuthProviderView{
			Name:     key,
			Label:    provider.Label,
			URL:      startURL,
			CSSClass: "btn-" + key,
		})
	}

	return views
}

// StartOAuth initiates the OAuth flow for a provider
func (h *AuthHandler) StartOAuth(w http.ResponseWriter, r *http.Request) {
	providerKey := r.PathValue("provider")
	provider, ok := h.oauthProviders[providerKey]
	if !ok || provider.Config == nil || provider.Config.ClientID == "" || provider.Config.ClientSecret == "" {
		h.httpError(w, r, "OAuth provider not configured", http.StatusBadRequest)
		return
	}

	state := utils.GenerateSessionID()
	nonce := utils.GenerateSessionID()

	h.setTempCookie(w, r, "oauth_state", state, 10*time.Minute)
	h.setTempCookie(w, r, "oauth_provider", providerKey, 10*time.Minute)
	h.setTempCookie(w, r, "oauth_nonce", nonce, 10*time.Minute)

	if familyCode := r.URL.Query().Get("family_code"); familyCode != "" {
		h.setTempCookie(w, r, "oauth_family_code", familyCode, 10*time.Minute)
	}

	redirectURL := h.oauthRedirectURL(r, providerKey)
	config := *provider.Config
	config.RedirectURL = redirectURL

	options := []oauth2.AuthCodeOption{oauth2.AccessTypeOnline}
	for key, value := range provider.AuthParams {
		options = append(options, oauth2.SetAuthURLParam(key, value))
	}
	if providerKey == "apple" {
		options = append(options, oauth2.SetAuthURLParam("nonce", nonce))
	}

	authURL := config.AuthCodeURL(state, options...)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// OAuthCallback handles the OAuth provider callback
func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	providerKey := r.PathValue("provider")
	provider, ok := h.oauthProviders[providerKey]
	if !ok || provider.Config == nil || provider.Config.ClientID == "" || provider.Config.ClientSecret == "" {
		h.httpError(w, r, "OAuth provider not configured", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if code == "" {
		h.httpError(w, r, "Missing authorization code", http.StatusBadRequest)
		return
	}

	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" || stateCookie.Value != state {
		h.httpError(w, r, "Invalid OAuth state", http.StatusBadRequest)
		return
	}
	if providerCookie, err := r.Cookie("oauth_provider"); err == nil {
		if providerCookie.Value != providerKey {
			h.httpError(w, r, "OAuth provider mismatch", http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	redirectURL := h.oauthRedirectURL(r, providerKey)
	config := *provider.Config
	config.RedirectURL = redirectURL

	token, err := config.Exchange(ctx, code)
	if err != nil {
		h.httpError(w, r, "Failed to exchange OAuth code", http.StatusBadRequest)
		return
	}

	userInfo, err := h.fetchOAuthUserInfo(ctx, providerKey, provider, token, r)
	if err != nil {
		h.httpError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	familyCode := ""
	if cookie, err := r.Cookie("oauth_family_code"); err == nil {
		familyCode = cookie.Value
	}

	// Clear temporary OAuth cookies
	h.clearTempCookie(w, r, "oauth_state")
	h.clearTempCookie(w, r, "oauth_provider")
	h.clearTempCookie(w, r, "oauth_nonce")
	h.clearTempCookie(w, r, "oauth_family_code")

	session, _, err := h.authService.OAuthLogin(providerKey, userInfo.Subject, userInfo.Email, userInfo.Name, familyCode)
	if err != nil {
		h.httpError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	http.SetCookie(w, utils.CreateSessionCookie(r, "session_id", session.ID, session.ExpiresAt))
	http.Redirect(w, r, "/parent/dashboard", http.StatusSeeOther)
}

func (h *AuthHandler) fetchOAuthUserInfo(ctx context.Context, providerKey string, provider OAuthProvider, token *oauth2.Token, r *http.Request) (oauthUserInfo, error) {
	switch providerKey {
	case "google":
		return h.fetchGoogleUser(ctx, provider, token)
	case "facebook":
		return h.fetchFacebookUser(ctx, provider, token)
	case "apple":
		return h.fetchAppleUser(ctx, provider, token, r)
	default:
		return oauthUserInfo{}, errors.New("unsupported OAuth provider")
	}
}

func (h *AuthHandler) fetchGoogleUser(ctx context.Context, provider OAuthProvider, token *oauth2.Token) (oauthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get(provider.UserInfoURL)
	if err != nil {
		return oauthUserInfo{}, fmt.Errorf("failed to fetch Google user info")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oauthUserInfo{}, fmt.Errorf("failed to fetch Google user info")
	}

	var payload struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return oauthUserInfo{}, fmt.Errorf("failed to parse Google user info")
	}

	return oauthUserInfo{Subject: payload.ID, Email: payload.Email, Name: payload.Name}, nil
}

func (h *AuthHandler) fetchFacebookUser(ctx context.Context, provider OAuthProvider, token *oauth2.Token) (oauthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get(provider.UserInfoURL)
	if err != nil {
		return oauthUserInfo{}, fmt.Errorf("failed to fetch Facebook user info")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oauthUserInfo{}, fmt.Errorf("failed to fetch Facebook user info")
	}

	var payload struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return oauthUserInfo{}, fmt.Errorf("failed to parse Facebook user info")
	}

	return oauthUserInfo{Subject: payload.ID, Email: payload.Email, Name: payload.Name}, nil
}

func (h *AuthHandler) fetchAppleUser(ctx context.Context, provider OAuthProvider, token *oauth2.Token, r *http.Request) (oauthUserInfo, error) {
	idToken, _ := token.Extra("id_token").(string)
	if idToken == "" {
		return oauthUserInfo{}, errors.New("missing Apple id_token")
	}

	nonce := ""
	if cookie, err := r.Cookie("oauth_nonce"); err == nil {
		nonce = cookie.Value
	}

	claims, err := parseAppleIDToken(ctx, idToken, provider.Config.ClientID, nonce)
	if err != nil {
		return oauthUserInfo{}, err
	}

	return oauthUserInfo{Subject: claims.Subject, Email: claims.Email, Name: claims.Name}, nil
}

func (h *AuthHandler) oauthRedirectURL(r *http.Request, providerKey string) string {
	baseURL := strings.TrimSpace(h.oauthRedirectBaseURL)
	if baseURL == "" {
		scheme := "http"
		if utils.IsSecureRequest(r) {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, r.Host)
	}
	return fmt.Sprintf("%s/auth/%s/callback", strings.TrimRight(baseURL, "/"), providerKey)
}

func (h *AuthHandler) setTempCookie(w http.ResponseWriter, r *http.Request, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   utils.IsSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func (h *AuthHandler) clearTempCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   utils.IsSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *AuthHandler) httpError(w http.ResponseWriter, r *http.Request, message string, status int) {
	data := map[string]interface{}{
		"Title":          "Login - WordClash",
		"Error":          message,
		"OAuthProviders": h.oauthProviderViews(r),
	}
	if err := h.templates.ExecuteTemplate(w, "login.tmpl", data); err != nil {
		http.Error(w, message, status)
	}
}

type appleTokenClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Nonce         string `json:"nonce"`
}

type appleJWK struct {
	Keys []appleJWKKey `json:"keys"`
}

type appleJWKKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type appleParsedClaims struct {
	Subject string
	Email   string
	Name    string
}

func parseAppleIDToken(ctx context.Context, idToken, clientID, nonce string) (appleParsedClaims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	claims := &appleTokenClaims{}

	parsedToken, err := parser.ParseWithClaims(idToken, claims, func(token *jwt.Token) (interface{}, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing key id")
		}
		return fetchApplePublicKey(ctx, kid)
	})
	if err != nil || !parsedToken.Valid {
		return appleParsedClaims{}, errors.New("invalid Apple token")
	}

	if claims.Issuer != "https://appleid.apple.com" {
		return appleParsedClaims{}, errors.New("invalid Apple issuer")
	}
	if !audienceContains(claims.Audience, clientID) {
		return appleParsedClaims{}, errors.New("invalid Apple audience")
	}
	if nonce != "" && claims.Nonce != "" && claims.Nonce != nonce {
		return appleParsedClaims{}, errors.New("invalid Apple nonce")
	}
	if claims.Email == "" {
		return appleParsedClaims{}, errors.New("Apple email not available")
	}

	return appleParsedClaims{
		Subject: claims.Subject,
		Email:   claims.Email,
		Name:    "",
	}, nil
}

func audienceContains(audience jwt.ClaimStrings, value string) bool {
	for _, entry := range audience {
		if entry == value {
			return true
		}
	}
	return false
}

func fetchApplePublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://appleid.apple.com/auth/keys", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch Apple public keys")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jwk appleJWK
	if err := json.Unmarshal(body, &jwk); err != nil {
		return nil, err
	}

	for _, key := range jwk.Keys {
		if key.Kid != kid {
			continue
		}
		if key.Kty != "RSA" {
			return nil, errors.New("unexpected key type")
		}
		modulusBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			return nil, err
		}
		exponentBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			return nil, err
		}
		exponent := 0
		for _, b := range exponentBytes {
			exponent = exponent*256 + int(b)
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(modulusBytes),
			E: exponent,
		}, nil
	}

	return nil, errors.New("Apple public key not found")
}
