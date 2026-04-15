package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yourusername/go-api-starter/tests/base"
)

// UserTestSuite covers the /auth and /users endpoints.
type UserTestSuite struct {
	base.BaseTestSuite
}

func TestUserTestSuite(t *testing.T) {
	base.RunTestSuite(t, new(UserTestSuite))
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func (suite *UserTestSuite) baseURL() string {
	return fmt.Sprintf("http://localhost:%s", suite.Port)
}

func (suite *UserTestSuite) post(path string, body interface{}) *http.Response {
	suite.T().Helper()
	b, err := json.Marshal(body)
	suite.Require().NoError(err)
	resp, err := http.Post(suite.baseURL()+path, "application/json", bytes.NewBuffer(b))
	suite.Require().NoError(err)
	return resp
}

func (suite *UserTestSuite) postAuth(path, token string, body interface{}) *http.Response {
	suite.T().Helper()
	b, err := json.Marshal(body)
	suite.Require().NoError(err)
	req, err := http.NewRequest(http.MethodPost, suite.baseURL()+path, bytes.NewBuffer(b))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	return resp
}

func (suite *UserTestSuite) getAuth(path, token string) *http.Response {
	suite.T().Helper()
	req, err := http.NewRequest(http.MethodGet, suite.baseURL()+path, nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	_ = resp.Body.Close()
	return out
}

// ─── tests ────────────────────────────────────────────────────────────────────

func (suite *UserTestSuite) TestRegisterAndLogin() {
	// Register
	resp := suite.post("/auth/register", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	})
	suite.Equal(http.StatusCreated, resp.StatusCode)
	body := decodeJSON(suite.T(), resp)
	suite.NotEmpty(body["access_token"])
	suite.NotEmpty(body["refresh_token"])

	// Login with correct credentials
	resp = suite.post("/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	suite.Equal(http.StatusOK, resp.StatusCode)
	body = decodeJSON(suite.T(), resp)
	suite.NotEmpty(body["access_token"])
}

func (suite *UserTestSuite) TestLoginWrongPassword() {
	suite.post("/auth/register", map[string]string{
		"email": "wrong@example.com", "password": "correct123",
	})

	resp := suite.post("/auth/login", map[string]string{
		"email": "wrong@example.com", "password": "badpassword",
	})
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *UserTestSuite) TestDuplicateRegistration() {
	suite.post("/auth/register", map[string]string{
		"email": "dup@example.com", "password": "password123",
	})
	resp := suite.post("/auth/register", map[string]string{
		"email": "dup@example.com", "password": "password123",
	})
	suite.Equal(http.StatusConflict, resp.StatusCode)
}

func (suite *UserTestSuite) TestGetMe() {
	regResp := suite.post("/auth/register", map[string]string{
		"email": "me@example.com", "password": "password123", "name": "Alice",
	})
	tokens := decodeJSON(suite.T(), regResp)
	token := tokens["access_token"].(string)

	resp := suite.getAuth("/users/me", token)
	suite.Equal(http.StatusOK, resp.StatusCode)
	body := decodeJSON(suite.T(), resp)
	suite.Equal("me@example.com", body["email"])
	suite.Equal("Alice", body["name"])
}

func (suite *UserTestSuite) TestGetMeUnauthorized() {
	resp := suite.getAuth("/users/me", "invalid-token")
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *UserTestSuite) TestRefreshToken() {
	regResp := suite.post("/auth/register", map[string]string{
		"email": "refresh@example.com", "password": "password123",
	})
	tokens := decodeJSON(suite.T(), regResp)
	refreshToken := tokens["refresh_token"].(string)

	resp := suite.post("/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
	suite.Equal(http.StatusOK, resp.StatusCode)
	newTokens := decodeJSON(suite.T(), resp)
	suite.NotEmpty(newTokens["access_token"])
}

func (suite *UserTestSuite) TestLogout() {
	regResp := suite.post("/auth/register", map[string]string{
		"email": "logout@example.com", "password": "password123",
	})
	tokens := decodeJSON(suite.T(), regResp)
	token := tokens["access_token"].(string)

	// Logout
	logoutResp := suite.postAuth("/auth/logout", token, nil)
	suite.Equal(http.StatusOK, logoutResp.StatusCode)

	// Token should no longer work
	resp := suite.getAuth("/users/me", token)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}
