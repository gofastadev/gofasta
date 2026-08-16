package auth

import (
	"testing"
	"time"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testJWTService() *JWTService {
	return NewJWTService(&config.AuthConfig{
		JWTSecret:          "test-secret-key-for-testing",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
	})
}

func expiredJWTService() *JWTService {
	return NewJWTService(&config.AuthConfig{
		JWTSecret:          "test-secret-key-for-testing",
		AccessTokenExpiry:  -1 * time.Second, // already expired
		RefreshTokenExpiry: -1 * time.Second,
	})
}

// ---------- NewJWTService ----------

func TestNewJWTService(t *testing.T) {
	svc := testJWTService()
	assert.NotNil(t, svc)
	assert.Equal(t, []byte("test-secret-key-for-testing"), svc.secret)
	assert.Equal(t, 15*time.Minute, svc.accessExpiry)
	assert.Equal(t, 24*time.Hour, svc.refreshExpiry)
}

// ---------- GenerateToken ----------

func TestGenerateToken_Success(t *testing.T) {
	svc := testJWTService()

	token, err := svc.GenerateToken("user-1", "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateToken_DifferentUsersProduceDifferentTokens(t *testing.T) {
	svc := testJWTService()

	t1, _ := svc.GenerateToken("user-1", "admin")
	t2, _ := svc.GenerateToken("user-2", "admin")
	assert.NotEqual(t, t1, t2)
}

// ---------- ValidateToken ----------

func TestValidateToken_Valid(t *testing.T) {
	svc := testJWTService()

	token, err := svc.GenerateToken("user-1", "editor")
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.SubjectID())
	assert.Equal(t, "editor", claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestValidateToken_Expired(t *testing.T) {
	svc := expiredJWTService()

	token, err := svc.GenerateToken("user-1", "admin")
	require.NoError(t, err)

	_, err = svc.ValidateToken(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidateToken_InvalidString(t *testing.T) {
	svc := testJWTService()

	_, err := svc.ValidateToken("not-a-valid-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidateToken_WrongSecret(t *testing.T) {
	svc1 := testJWTService()
	svc2 := NewJWTService(&config.AuthConfig{
		JWTSecret:          "different-secret-key",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
	})

	token, err := svc1.GenerateToken("user-1", "admin")
	require.NoError(t, err)

	_, err = svc2.ValidateToken(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidateToken_EmptyString(t *testing.T) {
	svc := testJWTService()

	_, err := svc.ValidateToken("")
	require.Error(t, err)
}

// ---------- GenerateRefreshToken ----------

func TestGenerateRefreshToken_Success(t *testing.T) {
	svc := testJWTService()

	token, err := svc.GenerateRefreshToken("user-1")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.SubjectID())
	assert.Empty(t, claims.Role) // refresh tokens don't include role
}

// ---------- RefreshToken ----------

func TestRefreshToken_Success(t *testing.T) {
	svc := testJWTService()

	refreshToken, err := svc.GenerateRefreshToken("user-1")
	require.NoError(t, err)

	newAccessToken, err := svc.RefreshToken(refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)

	// Validate the new access token
	claims, err := svc.ValidateToken(newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.SubjectID())
}

func TestRefreshToken_InvalidRefreshToken(t *testing.T) {
	svc := testJWTService()

	_, err := svc.RefreshToken("invalid-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestRefreshToken_ExpiredRefreshToken(t *testing.T) {
	svc := expiredJWTService()

	token, err := svc.GenerateRefreshToken("user-1")
	require.NoError(t, err)

	_, err = svc.RefreshToken(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

// ---------- Token round-trip ----------

func TestValidateToken_UnexpectedSigningMethod(t *testing.T) {
	svc := testJWTService()
	// Create a token signed with the "none" method (non-HMAC)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Role: "admin",
	})
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = svc.ValidateToken(tokenStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestTokenRoundTrip(t *testing.T) {
	svc := testJWTService()

	// Generate access token
	accessToken, err := svc.GenerateToken("user-42", "moderator")
	require.NoError(t, err)

	// Validate it
	claims, err := svc.ValidateToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, "user-42", claims.SubjectID())
	assert.Equal(t, "moderator", claims.Role)

	// Generate refresh token
	refreshToken, err := svc.GenerateRefreshToken("user-42")
	require.NoError(t, err)

	// Use refresh to get new access token
	newAccess, err := svc.RefreshToken(refreshToken)
	require.NoError(t, err)

	// Validate the new access token
	newClaims, err := svc.ValidateToken(newAccess)
	require.NoError(t, err)
	assert.Equal(t, "user-42", newClaims.SubjectID())
}

// ---------- issuer ----------

func issuerJWTService(issuer string) *JWTService {
	return NewJWTService(&config.AuthConfig{
		JWTSecret:          "test-secret-key-for-testing",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             issuer,
	})
}

func TestValidateToken_RejectsAnotherIssuersToken(t *testing.T) {
	// Every service in a fleet usually shares one signing secret, so the
	// signature alone does not say the token was minted for *this* service.
	// Without the issuer check, a token for the reporting tool opens payments.
	payments := issuerJWTService("payments")
	reporting := issuerJWTService("reporting")

	token, err := reporting.GenerateToken("user-1", "admin")
	require.NoError(t, err)

	_, err = payments.ValidateToken(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "issuer")

	// Its own issuer still passes.
	own, err := payments.GenerateToken("user-1", "admin")
	require.NoError(t, err)
	_, err = payments.ValidateToken(own)
	assert.NoError(t, err)
}

func TestValidateToken_NoIssuerConfiguredChecksNone(t *testing.T) {
	// Deployments that predate the field must keep working unchanged.
	unset := testJWTService()
	token, err := issuerJWTService("somebody-else").GenerateToken("user-1", "admin")
	require.NoError(t, err)

	claims, err := unset.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.SubjectID())
}

func TestGenerateToken_StampsIssuerAndSubject(t *testing.T) {
	svc := issuerJWTService("descholar")
	token, err := svc.GenerateToken("user-1", "admin")
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "descholar", claims.Issuer)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Equal(t, "user-1", claims.SubjectID())
}

func TestGenerateTokenWithRoles_RoundTrips(t *testing.T) {
	svc := testJWTService()
	token, err := svc.GenerateTokenWithRoles("user-1", "admin", []string{"facilitator", "learner"})
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, []string{"facilitator", "learner"}, claims.Roles)
	assert.True(t, claims.HasRole("learner"))
}

func TestValidateToken_RejectsNoneAlgorithm(t *testing.T) {
	// alg=none is the classic JWT forgery: strip the signature and claim the
	// token needs none. WithValidMethods pins HS256 before the keyfunc runs.
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, &Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "attacker"}}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = testJWTService().ValidateToken(unsigned)
	assert.Error(t, err)
}

// ---------- Issuer ----------

func TestIssuer_ReportsWhatIsEnforced(t *testing.T) {
	// Callers read this to decide whether tokens they mint elsewhere will be
	// accepted here. An issuer that is configured but not reported leads to a
	// caller stamping the wrong one, or none.
	assert.Equal(t, "solago", issuerJWTService("solago").Issuer())
}

func TestIssuer_EmptyMeansNoCheck(t *testing.T) {
	// "" is not a missing value to be filled in with a default — it is the
	// configuration that says the issuer claim is not checked at all.
	assert.Equal(t, "", testJWTService().Issuer())
}

// ---------- keyFunc ----------

func TestKeyFunc_SuppliesTheSecretForAnHMACToken(t *testing.T) {
	svc := testJWTService()

	key, err := svc.keyFunc(jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{}))

	require.NoError(t, err)
	assert.Equal(t, svc.secret, key)
}

// The alg-confusion attack: a token signed with the public half of an RSA key
// pair, presented as HMAC so that the public key — which the attacker also has
// — becomes the shared secret. ValidateToken pins HS256 through
// WithValidMethods before the keyfunc is reached, so this is the second of two
// independent refusals; a caller that builds its own parser without that option
// still gets this one.
func TestKeyFunc_RefusesToKeyANonHMACToken(t *testing.T) {
	for _, method := range []jwt.SigningMethod{
		jwt.SigningMethodRS256,
		jwt.SigningMethodES256,
		jwt.SigningMethodNone,
	} {
		t.Run(method.Alg(), func(t *testing.T) {
			key, err := testJWTService().keyFunc(jwt.NewWithClaims(method, &Claims{}))

			require.Error(t, err)
			assert.Nil(t, key, "no key may be handed to a non-HMAC token")
			assert.Contains(t, err.Error(), "unexpected signing method")
			assert.Contains(t, err.Error(), method.Alg(), "the error should name the algorithm presented")
		})
	}
}
