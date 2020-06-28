package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/jenkins/api"

	"github.com/jsdidierlaurent/echo-middleware/cache"
	"github.com/jsdidierlaurent/echo-middleware/cache/mocks"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	. "github.com/stretchr/testify/mock"
)

func initErrorEcho() (ctx echo.Context, res *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/error", nil)
	res = httptest.NewRecorder()
	ctx = e.NewContext(req, res)

	return
}

func TestHTTPError_404(t *testing.T) {
	// Init
	ctx, res := initErrorEcho()

	// Parameters
	err := echo.NewHTTPError(http.StatusNotFound, "not found")

	// Expected
	apiError := APIError{
		Code:    http.StatusNotFound,
		Message: "Not Found",
	}
	j, e := json.Marshal(apiError)
	assert.NoError(t, e, "unable to marshal tile")

	// Test
	HTTPErrorHandler(err, ctx)

	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, string(j), strings.TrimSpace(res.Body.String()))
}

func TestHTTPError_500(t *testing.T) {
	// Init
	ctx, res := initErrorEcho()

	// Parameters
	err := errors.New("boom")

	// Expected
	apiError := APIError{
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	}
	j, e := json.Marshal(apiError)
	assert.NoError(t, e, "unable to marshal tile")

	// Test
	HTTPErrorHandler(err, ctx)

	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, string(j), strings.TrimSpace(res.Body.String()))
}

func TestHTTPError_MonitororError_WithoutTile(t *testing.T) {
	// Init
	ctx, res := initErrorEcho()

	// Parameters
	err := &models.MonitororError{Err: errors.New("boom"), Message: "rly big boom"}

	// Expected
	apiError := APIError{
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	}
	j, e := json.Marshal(apiError)
	assert.NoError(t, e, "unable to marshal tile")

	// Test
	HTTPErrorHandler(err, ctx)

	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, string(j), strings.TrimSpace(res.Body.String()))
}

func TestHTTPError_MonitororError_WithTile(t *testing.T) {
	// Init
	ctx, res := initErrorEcho()

	// Parameters
	tile := models.NewTile(api.JenkinsBuildTileType)
	tile.Label = "test jenkins"
	err := &models.MonitororError{Err: errors.New("boom"), Tile: tile, Message: "rly big boom"}

	// Expected
	expected := models.NewTile(api.JenkinsBuildTileType)
	expected.Label = "test jenkins"
	expected.Status = models.FailedStatus
	expected.Message = "rly big boom"
	j, e := json.Marshal(expected)
	assert.NoError(t, e, "unable to marshal tile")

	// Test
	HTTPErrorHandler(err, ctx)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, string(j), strings.TrimSpace(res.Body.String()))
}

func TestHTTPError_MonitororError_Timeout_WithoutStore(t *testing.T) {
	// Init
	ctx, res := initErrorEcho()

	// Parameters
	tile := models.NewTile("TEST")
	err := &models.MonitororError{Err: context.DeadlineExceeded, Tile: tile}

	// Expected
	expectedTile := tile
	expectedTile.Status = models.WarningStatus
	expectedTile.Message = "timeout/host unreachable"
	j, e := json.Marshal(expectedTile)
	assert.NoError(t, e, "unable to marshal tile")

	// Test
	HTTPErrorHandler(err, ctx)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, string(j), strings.TrimSpace(res.Body.String()))
}

func TestHTTPError_MonitororError_Timeout_WithWrongStore(t *testing.T) {
	// Init
	ctx, res := initErrorEcho()
	ctx.Set(models.DownstreamStoreContextKey, "store")

	// Parameters
	tile := models.NewTile("TEST")
	err := &models.MonitororError{Err: context.DeadlineExceeded, Tile: tile}

	// Expected
	expectedTile := tile
	expectedTile.Status = models.WarningStatus
	expectedTile.Message = "timeout/host unreachable"
	j, e := json.Marshal(expectedTile)
	assert.NoError(t, e, "unable to marshal tile")

	// Test
	HTTPErrorHandler(err, ctx)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, string(j), strings.TrimSpace(res.Body.String()))
}

func TestHTTPError_MonitororError_Timeout_CacheMiss(t *testing.T) {
	// Init
	ctx, res := initErrorEcho()
	mockStore := new(mocks.Store)
	mockStore.On("Get", AnythingOfType("string"), Anything).Return(cache.ErrCacheMiss)
	ctx.Set(models.DownstreamStoreContextKey, mockStore)

	// Parameters
	tile := models.NewTile("TEST")
	err := &models.MonitororError{Err: context.DeadlineExceeded, Tile: tile}

	// Expected
	expectedTile := tile
	expectedTile.Status = models.WarningStatus
	expectedTile.Message = "timeout/host unreachable"
	j, e := json.Marshal(expectedTile)
	assert.NoError(t, e, "unable to marshal tile")

	// Test
	HTTPErrorHandler(err, ctx)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, string(j), strings.TrimSpace(res.Body.String()))
	mockStore.AssertNumberOfCalls(t, "Get", 1)
	mockStore.AssertExpectations(t)
}

func TestHTTPError_MonitororError_Timeout_Success(t *testing.T) {
	// Init
	ctx, res := initErrorEcho()

	status := http.StatusOK
	header := ctx.Request().Header
	header.Add("header", "true")
	body := "body"

	mockStore := new(mocks.Store)
	mockStore.
		On("Get", AnythingOfType("string"), AnythingOfType("*cache.ResponseCache")).
		Return(nil).
		Run(func(args Arguments) {
			arg := args.Get(1).(*cache.ResponseCache)
			arg.Data = []byte(body)
			arg.Header = header
			arg.Status = status
		})
	ctx.Set(models.DownstreamStoreContextKey, mockStore)

	// Parameters
	tile := models.NewTile("TEST")
	err := &models.MonitororError{Err: context.DeadlineExceeded, Tile: tile}

	// Test
	HTTPErrorHandler(err, ctx)
	header.Add("Timeout-Recover", "true")

	assert.Equal(t, status, res.Code)
	assert.Equal(t, header, res.Header())
	assert.Equal(t, body, res.Body.String())
	mockStore.AssertNumberOfCalls(t, "Get", 1)
	mockStore.AssertExpectations(t)
}
