package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QueerGlobal/hub-framework/api"
	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/QueerGlobal/hub-framework/example/recipe-app/golang/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootHandler_BlocksSmokePathsAgainstLiveHubWithoutEnv(t *testing.T) {
	t.Setenv("ENABLE_SMOKE_FORWARD", "")

	app := startRecipeApplication(t, recipeAppDir(t))
	defer app.Stop()

	handler := rootHandler{hub: app.PublicHandler}
	for _, path := range smokeForwardPathVariants {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code, "path %q", path)
	}
}

func TestSmokeForward_NotRegisteredWhenEnvDisabled(t *testing.T) {
	t.Setenv("ENABLE_SMOKE_FORWARD", "")

	appHome := recipeAppDir(t)
	app := startRecipeApplication(t, appHome)
	defer app.Stop()

	hub := app.Hub.(*entity.Hub)
	_, ok := hub.GetService("smokeApp", "forward")
	assert.False(t, ok)
}

func TestSmokeForward_ForwardsToUpstreamWhenEnabled(t *testing.T) {
	t.Setenv("ENABLE_SMOKE_FORWARD", "true")

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasPrefix(r.URL.Path, "/apply/exampleTaskPython"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ingredients":["a little extra love"]}`))
	}))
	defer upstream.Close()

	appHome := prepareRecipeAppHome(t, upstream.URL)
	app := startRecipeApplication(t, appHome)
	defer app.Stop()

	require.NotNil(t, app.PublicHandler)

	payload := `{"name":"Smoke Cake","ingredients":["flour"],"steps":["mix"],"comments":[]}`
	req := httptest.NewRequest(http.MethodPost, "/smokeApp/forward", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.PublicHandler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body, err := io.ReadAll(rec.Result().Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "a little extra love")
}

func TestSmokeForward_ComposeIntegration(t *testing.T) {
	hubURL := os.Getenv("HUB_URL")
	if hubURL == "" {
		t.Skip("set HUB_URL when hub-app is running (e.g. docker compose up)")
	}

	payload := `{"name":"Smoke Cake","ingredients":["flour"],"steps":["mix"],"comments":[]}`
	req, err := http.NewRequest(http.MethodPost, hubURL+"/smokeApp/forward", strings.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "a little extra love")
}

func startRecipeApplication(t *testing.T, appHome string) *api.Application {
	exampleTaskConstructor := api.TaskConstructorFunc(func(config map[string]any) (api.Task, error) {
		return tasks.NewExampleTaskGolang(config), nil
	})
	api.RegisterTaskType("exampleTaskGolang", exampleTaskConstructor)

	app := api.NewApplication(
		"exampleApp",
		api.WithLogLevel(api.InfoLevel),
		api.WithApplicationHome(appHome),
		api.WithHTTPListen(false),
	)
	require.NotNil(t, app)
	require.NoError(t, app.Start())
	return app
}

func recipeAppDir(t *testing.T) string {
	candidates := []string{
		".",
		"example/recipe-app",
		filepath.Join("..", "example", "recipe-app"),
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "hub.yaml")); err == nil {
			abs, err := filepath.Abs(dir)
			require.NoError(t, err)
			return abs
		}
	}
	t.Fatal("recipe-app directory not found")
	return ""
}

func prepareRecipeAppHome(t *testing.T, upstreamURL string) string {
	src := recipeAppDir(t)
	dst := t.TempDir()
	require.NoError(t, copyDir(src, dst))

	path := filepath.Join(dst, "aggregates", "smoke-forward.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	updated := strings.ReplaceAll(string(data), "http://python-handler:5000", strings.TrimRight(upstreamURL, "/"))
	require.NoError(t, os.WriteFile(path, []byte(updated), 0644))
	return dst
}

func copyDir(src, dst string) error {
	return copyDirWalk(src, dst, src)
}

func copyDirWalk(src, dst, root string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDirWalk(srcPath, dstPath, root); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return err
		}
	}
	return nil
}
