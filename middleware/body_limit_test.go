package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestBodyLimit(t *testing.T) {
	e := echo.New()
	hw := []byte("Hello, World!")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(hw))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, string(body))
	}

	// Based on content length (within limit)
	if assert.NoError(t, BodyLimit("2M")(h)(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, hw, rec.Body.Bytes())
	}

	// Based on content length (overlimit)
	he := BodyLimit("2B")(h)(c).(*echo.HTTPError)
	assert.Equal(t, http.StatusRequestEntityTooLarge, he.Code)

	// Based on content read (within limit)
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(hw))
	req.ContentLength = -1
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	if assert.NoError(t, BodyLimit("2M")(h)(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Hello, World!", rec.Body.String())
	}

	// Based on content read (overlimit)
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(hw))
	req.ContentLength = -1
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	he = BodyLimit("2B")(h)(c).(*echo.HTTPError)
	assert.Equal(t, http.StatusRequestEntityTooLarge, he.Code)
}

func TestBodyLimitReader(t *testing.T) {
	hw := []byte("Hello, World!")
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(hw))
	rec := httptest.NewRecorder()

	config := BodyLimitConfig{
		Skipper: DefaultSkipper,
		Limit:   "2B",
		limit:   2,
	}
	reader := &limitedReader{
		BodyLimitConfig: config,
		reader:          io.NopCloser(bytes.NewReader(hw)),
		context:         e.NewContext(req, rec),
	}

	// read all should return ErrStatusRequestEntityTooLarge
	_, err := io.ReadAll(reader)
	he := err.(*echo.HTTPError)
	assert.Equal(t, http.StatusRequestEntityTooLarge, he.Code)

	// reset reader and read two bytes must succeed
	bt := make([]byte, 2)
	reader.Reset(io.NopCloser(bytes.NewReader(hw)), e.NewContext(req, rec))
	n, err := reader.Read(bt)
	assert.Equal(t, 2, n)
	assert.Equal(t, nil, err)
}
