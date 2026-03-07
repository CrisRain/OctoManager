package service

import (
	"context"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/email/outlookoauth"
)

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func (s *emailAccountService) BuildOutlookAuthorizeURL(
	ctx context.Context,
	req *dto.OutlookAuthorizeURLRequest,
) (*dto.OutlookAuthorizeURLResponse, error) {
	_ = ctx
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	clientID := trim(req.ClientID)
	redirectURI := trim(req.RedirectURI)
	if clientID == "" {
		return nil, invalidInput("client_id is required")
	}
	if redirectURI == "" {
		return nil, invalidInput("redirect_uri is required")
	}

	urlValue, err := outlookoauth.BuildAuthorizeURL(outlookoauth.AuthorizeURLInput{
		Tenant:              normalizeOutlookTenant(req.Tenant),
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scope:               normalizeScopeList(req.Scope),
		State:               trim(req.State),
		Prompt:              trim(req.Prompt),
		LoginHint:           trim(req.LoginHint),
		CodeChallenge:       trim(req.CodeChallenge),
		CodeChallengeMethod: trim(req.CodeChallengeMethod),
	})
	if err != nil {
		return nil, invalidInput("failed to build outlook authorize url: " + err.Error())
	}
	codeChallengeMethod := trim(req.CodeChallengeMethod)
	if trim(req.CodeChallenge) != "" && codeChallengeMethod == "" {
		codeChallengeMethod = "S256"
	}

	return &dto.OutlookAuthorizeURLResponse{
		AuthorizeURL:        urlValue,
		Tenant:              normalizeOutlookTenant(req.Tenant),
		Scope:               normalizeScopeList(req.Scope),
		State:               trim(req.State),
		CodeChallenge:       trim(req.CodeChallenge),
		CodeChallengeMethod: codeChallengeMethod,
	}, nil
}

func (s *emailAccountService) ExchangeOutlookCode(
	ctx context.Context,
	req *dto.OutlookExchangeCodeRequest,
) (*dto.OutlookTokenResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	response, err := outlookoauth.ExchangeCode(ctx, outlookoauth.ExchangeCodeInput{
		Tenant:       normalizeOutlookTenant(req.Tenant),
		ClientID:     trim(req.ClientID),
		ClientSecret: trim(req.ClientSecret),
		RedirectURI:  trim(req.RedirectURI),
		Code:         trim(req.Code),
		Scope:        normalizeScopeList(req.Scope),
		TokenURL:     trim(req.TokenURL),
		CodeVerifier: trim(req.CodeVerifier),
	})
	if err != nil {
		return nil, invalidInput("failed to exchange outlook oauth code: " + err.Error())
	}
	return buildOutlookTokenResponse(response), nil
}

func (s *emailAccountService) RefreshOutlookToken(
	ctx context.Context,
	req *dto.OutlookRefreshTokenRequest,
) (*dto.OutlookTokenResponse, error) {
	if req == nil {
		return nil, invalidInput("payload is required")
	}
	response, err := outlookoauth.RefreshToken(ctx, outlookoauth.RefreshTokenInput{
		Tenant:       normalizeOutlookTenant(req.Tenant),
		ClientID:     trim(req.ClientID),
		ClientSecret: trim(req.ClientSecret),
		RefreshToken: trim(req.RefreshToken),
		Scope:        normalizeScopeList(req.Scope),
		TokenURL:     trim(req.TokenURL),
	})
	if err != nil {
		return nil, invalidInput("failed to refresh outlook oauth token: " + err.Error())
	}
	return buildOutlookTokenResponse(response), nil
}

func buildOutlookTokenResponse(response outlookoauth.TokenResponse) *dto.OutlookTokenResponse {
	return &dto.OutlookTokenResponse{
		TokenType:    response.TokenType,
		Scope:        response.Scope,
		ExpiresIn:    response.ExpiresIn,
		ExpiresAt:    response.ExpiresAt.UTC().Format(timeRFC3339),
		TokenURL:     response.TokenURL,
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
	}
}
