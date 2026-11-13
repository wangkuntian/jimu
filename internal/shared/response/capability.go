package response

import appErrs "jimu/internal/shared/errors"

var capabilityCodeToKey = map[int]string{
	appErrs.CodeUserNotFound:               "user_not_found",
	appErrs.CodeUserExists:                 "user_exists",
	appErrs.CodeInvalidPassword:            "invalid_credentials",
	appErrs.CodeRoleNotFound:               "role_not_found",
	appErrs.CodeInvalidResetCode:           "invalid_reset_code",
	appErrs.CodeMFARequired:                "mfa_required",
	appErrs.CodeInvalidMFA:                 "invalid_mfa",
	appErrs.CodePasswordReused:             "password_reused",
	appErrs.CodePasswordBreached:           "password_breached",
	appErrs.CodeWebAuthnNoCredential:       "webauthn_no_credential",
	appErrs.CodeWebAuthnVerificationFailed: "webauthn_verification_failed",
	appErrs.CodeOAuthProviderNotFound:      "oauth_provider_not_found",
	appErrs.CodeCaptchaRequired:            "captcha_required",
	appErrs.CodeCaptchaInvalid:             "captcha_invalid",
	appErrs.CodeTenantNotFound:             "tenant_not_found",
	appErrs.CodeTenantExists:               "tenant_exists",
	appErrs.CodeTenantProtected:            "tenant_protected",
	appErrs.CodeTenantCodeFormat:           "tenant_code_format",
	appErrs.CodeQuotaExceeded:              "quota_exceeded",
	appErrs.CodePlanNotFound:               "plan_not_found",
}

func init() {
	for code, key := range capabilityCodeToKey {
		codeToKey[code] = key
	}
}
