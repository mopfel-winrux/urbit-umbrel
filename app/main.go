package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

//go:embed ui/dist/**
var embedded embed.FS
var webFS fs.FS

const (
	sessKey  = "umbrel_user"
	urbitBin = "/usr/bin/urbit"
)

var (
	keyDir              = env("KEY_DIR", "/data/keys")
	pierDir             = env("PIER_DIR", "/data/piers")
	appPort             = env("APP_PORT", "8090")
	amesPort            = envInt("AMES_PORT", 34343)
	defaultLoom         = envInt("DEFAULT_LOOM", 31)
	loopback            = "http://127.0.0.1:12321"
	appPwd              = env("APP_PASSWORD", "")
	minMigrationVersion = semVersion{Major: 3, Minor: 5}
	versionRE           = regexp.MustCompile(`([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
	releases            = newVereCatalog()
	runnerErrStopped    = errors.New("runner stopped")

	urbitMu sync.Mutex
	urbit   *runner
)

type runner struct {
	cmd      *exec.Cmd
	buf      bytes.Buffer
	mu       sync.Mutex
	started  time.Time
	stopOnce sync.Once
	quit     chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

type semVersion struct {
	Major int
	Minor int
	Patch int
}

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type vereRelease struct {
	Tag         string `json:"tag"`
	Version     string `json:"version"`
	DownloadURL string `json:"-"`
	parsed      semVersion
}

type vereCatalog struct {
	mu         sync.RWMutex
	refreshMu  sync.Mutex
	versions   []vereRelease
	currentTag string
	current    semVersion
	err        string
	loadedAt   time.Time
}

func init() {
	s, _ := fs.Sub(embedded, "ui/dist")
	webFS = s
	if err := os.MkdirAll(keyDir, 0o755); err != nil {
		panic("Must have a key dir to function")
	}
	if err := os.MkdirAll(pierDir, 0o755); err != nil {
		panic("Must have a pier dir to function")
	}
}

func main() {
	loadCtx, cancelLoad := context.WithTimeout(context.Background(), 12*time.Second)
	if err := releases.refresh(loadCtx); err != nil {
		log.Println("vere release catalog error:", err)
	}
	cancelLoad()

	shipURL, _ := url.Parse("http://127.0.0.1:8000")
	shipProxy := httputil.NewSingleHostReverseProxy(shipURL)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/api/logs", "/api/status"}}), gin.Recovery())

	store := cookie.NewStore(make([]byte, 64))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   99999,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("umbrel", store))

	api := r.Group("/api")
	api.POST("/login", login)
	api.POST("/logout", logout)

	authed := api.Group("/")
	authed.Use(authRequired())
	{
		authed.GET("/status", status)
		authed.GET("/logs", logs)
		authed.GET("/migration-options", migrationOptions)
		authed.POST("/boot", bootExisting)
		authed.POST("/boot-comet", bootComet)
		authed.POST("/migrate", migrateVere)
		authed.POST("/stop", stopUrbit)
		authed.POST("/upload-key", uploadKey)
		authed.POST("/upload-pier", uploadPier)
	}

	r.StaticFS("/static", http.FS(webFS))

	r.GET("/launch", func(c *gin.Context) {
		data, _ := fs.ReadFile(webFS, "index.html")
		c.Data(200, "text/html; charset=utf-8", data)
	})
	r.GET("/launch/*any", func(c *gin.Context) {
		data, _ := fs.ReadFile(webFS, "index.html")
		c.Data(200, "text/html; charset=utf-8", data)
	})

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") ||
			strings.HasPrefix(path, "/static") ||
			strings.HasPrefix(path, "/launch") {
			c.AbortWithStatus(404)
			return
		}
		if active := currentRunner(); active != nil {
			if up, _, _, _ := active.snap(); up {
				defer func() {
					if rec := recover(); rec != nil {
						log.Printf("proxy panic: %v", rec)
					}
				}()
				shipProxy.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		data, _ := fs.ReadFile(webFS, "index.html")
		c.Data(200, "text/html; charset=utf-8", data)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", appPort),
		Handler: r,
	}

	sigCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	go func() {
		<-sigCtx.Done()
		stopActiveRunner()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newRunner(run func(*runner) error) *runner {
	ctx, cancel := context.WithCancel(context.Background())
	r := &runner{
		started: time.Now(),
		quit:    make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
	go func() {
		defer close(r.quit)
		if err := run(r); err != nil && !errors.Is(err, runnerErrStopped) && !errors.Is(err, context.Canceled) {
			r.logf("Runner error: %v", err)
		}
	}()
	return r
}

func (r *runner) execStep(bin string, args ...string) error {
	if err := r.ctx.Err(); err != nil {
		return runnerErrStopped
	}

	cmd := exec.Command(bin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	r.logf("$ %s %s", filepath.Base(bin), strings.Join(args, " "))

	if err := cmd.Start(); err != nil {
		return err
	}
	r.setCmd(cmd)

	go r.tail(stdout)
	go r.tail(stderr)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		r.clearCmd(cmd)
		if errors.Is(r.ctx.Err(), context.Canceled) {
			return runnerErrStopped
		}
		return err
	case <-r.ctx.Done():
		r.signal(cmd, syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			r.signal(cmd, syscall.SIGKILL)
			<-done
		}
		r.clearCmd(cmd)
		return runnerErrStopped
	}
}

func (r *runner) stop() {
	if r == nil {
		return
	}

	r.stopOnce.Do(r.cancel)

	r.mu.Lock()
	cmd := r.cmd
	r.mu.Unlock()

	if cmd != nil {
		r.signal(cmd, syscall.SIGTERM)
	}

	select {
	case <-r.quit:
		return
	case <-time.After(3 * time.Second):
		if cmd != nil {
			r.signal(cmd, syscall.SIGKILL)
		}
		select {
		case <-r.quit:
		case <-time.After(2 * time.Second):
		}
	}
}

func (r *runner) snap() (up bool, upFor time.Duration, tail string, err error) {
	r.mu.Lock()
	tail = r.buf.String()
	started := r.started
	r.mu.Unlock()

	select {
	case <-r.quit:
		return false, 0, tail, nil
	default:
		return true, time.Since(started), tail, nil
	}
}

func (r *runner) tail(rc io.ReadCloser) {
	defer rc.Close()

	sc := bufio.NewScanner(rc)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		r.appendLog(sc.Text())
	}
	if err := sc.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
		r.logf("log stream error: %v", err)
	}
}

func (r *runner) setCmd(cmd *exec.Cmd) {
	r.mu.Lock()
	r.cmd = cmd
	r.mu.Unlock()
}

func (r *runner) clearCmd(cmd *exec.Cmd) {
	r.mu.Lock()
	if r.cmd == cmd {
		r.cmd = nil
	}
	r.mu.Unlock()
}

func (r *runner) signal(cmd *exec.Cmd, sig syscall.Signal) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	if pid <= 0 {
		return
	}
	if err := syscall.Kill(-pid, sig); err != nil {
		_ = syscall.Kill(pid, sig)
	}
}

func (r *runner) appendLog(line string) {
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	r.mu.Lock()
	r.buf.WriteString(line)
	if r.buf.Len() > 32*1024 {
		r.buf.Next(r.buf.Len() - 32*1024)
	}
	r.mu.Unlock()
	_, _ = os.Stdout.Write([]byte(line))
}

func (r *runner) logf(format string, args ...any) {
	r.appendLog(fmt.Sprintf(format, args...))
}

func newVereCatalog() *vereCatalog {
	return &vereCatalog{}
}

func (vc *vereCatalog) refresh(ctx context.Context) error {
	vc.refreshMu.Lock()
	defer vc.refreshMu.Unlock()

	currentTag, currentVersion, err := detectCurrentVere()
	if err != nil {
		vc.setState(nil, currentTag, currentVersion, err.Error())
		return err
	}

	assetName, err := releaseAssetName()
	if err != nil {
		vc.setState(nil, currentTag, currentVersion, err.Error())
		return err
	}

	ghReleases, err := fetchGithubReleases(ctx)
	if err != nil {
		vc.setState(nil, currentTag, currentVersion, err.Error())
		return err
	}

	versions := make([]vereRelease, 0, len(ghReleases))
	for _, rel := range ghReleases {
		if rel.Draft || rel.Prerelease {
			continue
		}
		parsed, ok := parseTagVersion(rel.TagName)
		if !ok {
			continue
		}
		if parsed.Compare(minMigrationVersion) < 0 || parsed.Compare(currentVersion) > 0 {
			continue
		}

		assetURL := ""
		for _, asset := range rel.Assets {
			if asset.Name == assetName {
				assetURL = asset.BrowserDownloadURL
				break
			}
		}
		if assetURL == "" && rel.TagName != currentTag {
			continue
		}

		versions = append(versions, vereRelease{
			Tag:         rel.TagName,
			Version:     parsed.String(),
			DownloadURL: assetURL,
			parsed:      parsed,
		})
	}

	errMsg := ""
	if len(versions) == 0 {
		errMsg = "no compatible vere releases found"
	}
	vc.setState(versions, currentTag, currentVersion, errMsg)
	if errMsg != "" {
		return errors.New(errMsg)
	}
	return nil
}

func (vc *vereCatalog) setState(versions []vereRelease, currentTag string, current semVersion, errMsg string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	vc.versions = append([]vereRelease(nil), versions...)
	vc.currentTag = currentTag
	vc.current = current
	vc.err = errMsg
	vc.loadedAt = time.Now()
}

func (vc *vereCatalog) snapshot() ([]vereRelease, string, string, string) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	versions := append([]vereRelease(nil), vc.versions...)
	return versions, vc.currentTag, vc.current.String(), vc.err
}

func (vc *vereCatalog) migrationPath(startTag string) ([]vereRelease, string, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	if len(vc.versions) == 0 {
		if vc.err != "" {
			return nil, vc.currentTag, errors.New(vc.err)
		}
		return nil, vc.currentTag, errors.New("no migration versions loaded")
	}

	startIdx := -1
	for i, rel := range vc.versions {
		if rel.Tag == startTag {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		if startTag == vc.currentTag && vc.currentTag != "" {
			return nil, vc.currentTag, nil
		}
		return nil, vc.currentTag, fmt.Errorf("unknown starting version %q", startTag)
	}

	begin := 0
	if vc.versions[0].Tag == vc.currentTag {
		begin = 1
	}
	if startIdx < begin {
		return nil, vc.currentTag, nil
	}

	steps := append([]vereRelease(nil), vc.versions[begin:startIdx+1]...)
	for left, right := 0, len(steps)-1; left < right; left, right = left+1, right-1 {
		steps[left], steps[right] = steps[right], steps[left]
	}
	return steps, vc.currentTag, nil
}

func fetchGithubReleases(ctx context.Context) ([]githubRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	all := make([]githubRelease, 0, 64)

	for page := 1; page <= 4; page++ {
		url := fmt.Sprintf("https://api.github.com/repos/urbit/vere/releases?per_page=100&page=%d", page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "urbit-umbrel")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			return nil, fmt.Errorf("github releases request failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
		}

		var pageReleases []githubRelease
		err = json.NewDecoder(resp.Body).Decode(&pageReleases)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		all = append(all, pageReleases...)
		if len(pageReleases) < 100 {
			break
		}
	}

	return all, nil
}

func detectCurrentVere() (string, semVersion, error) {
	out, err := exec.Command(urbitBin, "-R").CombinedOutput()
	match := versionRE.FindStringSubmatch(string(out))
	if len(match) < 2 {
		return "", semVersion{}, fmt.Errorf("could not parse current vere version from %q", strings.TrimSpace(string(out)))
	}

	version, ok := parseSemVersion(match[1])
	if !ok {
		return "", semVersion{}, fmt.Errorf("invalid current vere version %q", match[1])
	}

	if err != nil {
		return "vere-v" + version.String(), version, nil
	}
	return "vere-v" + version.String(), version, nil
}

func releaseAssetName() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "linux-x86_64.tgz", nil
	case "arm64":
		return "linux-aarch64.tgz", nil
	default:
		return "", fmt.Errorf("unsupported architecture %q", runtime.GOARCH)
	}
}

func parseSemVersion(raw string) (semVersion, bool) {
	parts := strings.Split(raw, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return semVersion{}, false
	}

	vals := make([]int, 3)
	for i := range parts {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return semVersion{}, false
		}
		vals[i] = n
	}

	return semVersion{
		Major: vals[0],
		Minor: vals[1],
		Patch: vals[2],
	}, true
}

func parseTagVersion(tag string) (semVersion, bool) {
	cleaned := strings.TrimPrefix(tag, "vere-")
	cleaned = strings.TrimPrefix(cleaned, "v")
	cleaned = strings.SplitN(cleaned, "-", 2)[0]
	return parseSemVersion(cleaned)
}

func (v semVersion) Compare(other semVersion) int {
	switch {
	case v.Major != other.Major:
		if v.Major < other.Major {
			return -1
		}
	case v.Minor != other.Minor:
		if v.Minor < other.Minor {
			return -1
		}
	case v.Patch != other.Patch:
		if v.Patch < other.Patch {
			return -1
		}
	default:
		return 0
	}
	return 1
}

func (v semVersion) String() string {
	if v == (semVersion{}) {
		return ""
	}
	if v.Patch == 0 {
		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sessions.Default(c).Get(sessKey) != "umbrel" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

func login(c *gin.Context) {
	var in struct{ User, Pass string }
	if c.BindJSON(&in) != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if in.User == "umbrel" && in.Pass == appPwd {
		s := sessions.Default(c)
		s.Set(sessKey, "umbrel")
		s.Save()
		c.Header("Content-Type", "application/json")
		c.JSON(200, gin.H{"status": "success"})
		return
	}
	c.Status(401)
}

func logout(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Save()
	c.Status(200)
}

func status(c *gin.Context) {
	running := false
	var upFor float64

	if active := currentRunner(); active != nil {
		up, dur, _, _ := active.snap()
		running = up
		upFor = dur.Seconds()
	}

	var code *json.RawMessage
	state := "stopped"
	if running {
		code = getCode()
		if code == nil {
			state = "booting"
		} else {
			state = "running"
		}
	}

	entries, _ := os.ReadDir(pierDir)
	var piers []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(pierDir, e.Name())
		piers = append(piers, dir+"/")
	}

	c.JSON(200, gin.H{
		"keys":         glob(keyDir, "*.key"),
		"piers":        piers,
		"loomValues":   []int{31, 32, 33},
		"code":         code,
		"urbitRunning": running,
		"uptime":       upFor,
		"state":        state,
	})
}

func logs(c *gin.Context) {
	active := currentRunner()
	if active == nil {
		c.JSON(200, gin.H{"running": false})
		return
	}
	up, dur, tail, err := active.snap()
	c.JSON(200, gin.H{
		"running": up,
		"uptime":  int(dur.Seconds()),
		"tail":    tail,
		"error":   fmt.Sprint(err),
	})
}

func migrationOptions(c *gin.Context) {
	if versions, _, _, _ := releases.snapshot(); len(versions) == 0 {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		_ = releases.refresh(ctx)
		cancel()
	}

	versions, currentTag, currentVersion, errMsg := releases.snapshot()
	c.JSON(200, gin.H{
		"versions":       versions,
		"currentTag":     currentTag,
		"currentVersion": currentVersion,
		"error":          errMsg,
	})
}

func stopUrbit(c *gin.Context) {
	stopActiveRunner()
	c.Status(202)
}

func currentRunner() *runner {
	urbitMu.Lock()
	defer urbitMu.Unlock()
	return urbit
}

func stopActiveRunner() {
	urbitMu.Lock()
	active := urbit
	urbit = nil
	urbitMu.Unlock()

	if active != nil {
		active.stop()
	}
}

func setActiveRunner(r *runner) {
	urbitMu.Lock()
	urbit = r
	urbitMu.Unlock()
}

func startUrbit(args []string) {
	setActiveRunner(newRunner(func(r *runner) error {
		return r.execStep(urbitBin, args...)
	}))
}

func startMigration(pier string, loom int, startTag string) {
	setActiveRunner(newRunner(func(r *runner) error {
		return runMigration(r, pier, loom, startTag)
	}))
}

func bootExisting(c *gin.Context) {
	stopActiveRunner()

	var req struct {
		Path string
		Loom int
	}
	if c.BindJSON(&req) != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	loom := pick(req.Loom, defaultLoom)

	switch {
	case strings.HasSuffix(req.Path, ".key"):
		ship := strings.TrimSuffix(filepath.Base(req.Path), ".key")
		pier := filepath.Join(pierDir, ship)
		startUrbit(buildKeyBootArgs(ship, req.Path, pier, loom))
		go removeKeyEventually(req.Path)

	default:
		pier, err := resolvePierPath(req.Path)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		startUrbit(buildPierBootArgs(pier, loom))
	}

	c.Status(202)
}

func migrateVere(c *gin.Context) {
	stopActiveRunner()

	var req struct {
		Path    string `json:"path"`
		Loom    int    `json:"loom"`
		Version string `json:"version"`
	}
	if c.BindJSON(&req) != nil || req.Version == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	pier, err := resolvePierPath(req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, ok := parseTagVersion(req.Version); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid starting version"})
		return
	}

	startMigration(pier, pick(req.Loom, defaultLoom), req.Version)
	c.Status(202)
}

func bootComet(c *gin.Context) {
	stopActiveRunner()

	var req struct{ Loom int }
	_ = c.BindJSON(&req)

	rand.Seed(time.Now().UnixNano())
	name := fmt.Sprintf("comet-%d", rand.Intn(1<<16))
	pier := filepath.Join(pierDir, name)

	startUrbit(buildCometBootArgs(pier, pick(req.Loom, defaultLoom)))
	c.Status(202)
}

func runMigration(r *runner, pier string, loom int, startTag string) error {
	steps, currentTag, err := releases.migrationPath(startTag)
	if err != nil {
		return err
	}

	r.logf("Starting vere migration replay on %s from %s", filepath.Base(pier), startTag)

	tmpDir, err := os.MkdirTemp("", "vere-migrate-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cache := make(map[string]string)
	replayArgs := buildReplayArgs(pier, loom)

	fetch := func(rel vereRelease) (string, error) {
		if bin, ok := cache[rel.Tag]; ok {
			return bin, nil
		}
		bin, err := downloadReleaseBinary(r.ctx, rel, tmpDir)
		if err != nil {
			return "", err
		}
		cache[rel.Tag] = bin
		return bin, nil
	}

	for _, rel := range steps {
		if errors.Is(r.ctx.Err(), context.Canceled) {
			return runnerErrStopped
		}

		bin, err := fetch(rel)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return runnerErrStopped
			}
			r.logf("Skipping %s: %v", rel.Tag, err)
			continue
		}

		r.logf("Replaying event log with %s", rel.Tag)
		if err := r.execStep(bin, replayArgs...); err != nil {
			if errors.Is(err, runnerErrStopped) {
				return err
			}
			r.logf("Replay step %s failed: %v. Continuing.", rel.Tag, err)
		}
	}

	if currentTag == "" {
		currentTag = "current runtime"
	}
	r.logf("Replaying event log with %s", currentTag)
	if err := r.execStep(urbitBin, replayArgs...); err != nil {
		if errors.Is(err, runnerErrStopped) {
			return err
		}
		r.logf("Current runtime replay failed: %v. Attempting normal boot anyway.", err)
	}

	r.logf("Starting bundled runtime after replay")
	return r.execStep(urbitBin, buildPierBootArgs(pier, loom)...)
}

func downloadReleaseBinary(ctx context.Context, rel vereRelease, dir string) (string, error) {
	if rel.DownloadURL == "" {
		return "", errors.New("release asset unavailable for this architecture")
	}

	target := filepath.Join(dir, rel.Tag)
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		return target, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.DownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "urbit-umbrel")

	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("download failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		switch {
		case errors.Is(err, io.EOF):
			return "", errors.New("vere binary not found in release archive")
		case err != nil:
			return "", err
		case hdr.Typeflag != tar.TypeReg:
			continue
		}

		name := filepath.Base(hdr.Name)
		if !strings.HasPrefix(name, "vere-") {
			continue
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return target, nil
	}
}

func buildReplayArgs(pier string, loom int) []string {
	return []string{
		"-Lxt",
		"--loom", strconv.Itoa(loom),
		pier,
	}
}

func buildPierBootArgs(pier string, loom int) []string {
	return []string{
		"-t",
		"-b", "0.0.0.0",
		"--http-port", "8000",
		"-p", strconv.Itoa(amesPort),
		"--loom", strconv.Itoa(loom),
		pier,
	}
}

func buildKeyBootArgs(ship, keyPath, pier string, loom int) []string {
	return []string{
		"-t",
		"--http-port", "8000",
		"-b", "0.0.0.0",
		"-p", strconv.Itoa(amesPort),
		"--loom", strconv.Itoa(loom),
		"-w", ship,
		"-k", keyPath,
		"-c", pier,
	}
}

func buildCometBootArgs(pier string, loom int) []string {
	return []string{
		"-t",
		"-p", strconv.Itoa(amesPort),
		"--http-port", "8000",
		"--loom", strconv.Itoa(loom),
		"-c", pier,
	}
}

func removeKeyEventually(keyPath string) {
	for i := 0; i < 5; i++ {
		time.Sleep(30 * time.Second)
		if err := os.Remove(keyPath); err == nil {
			return
		}
	}
}

func resolvePierPath(raw string) (string, error) {
	cleanBase := filepath.Clean(pierDir)
	cleanRaw := filepath.Clean(strings.TrimSuffix(raw, string(os.PathSeparator)))
	if !strings.HasPrefix(cleanRaw, cleanBase+string(os.PathSeparator)) {
		return "", errors.New("select an existing pier to replay")
	}

	pier := filepath.Join(pierDir, filepath.Base(cleanRaw))
	info, err := os.Stat(pier)
	if err != nil {
		return "", fmt.Errorf("pier not found: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("selected path is not a pier")
	}
	return pier, nil
}

func uploadKey(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		c.Status(400)
		return
	}
	dst := filepath.Join(keyDir, filepath.Base(f.Filename))
	if c.SaveUploadedFile(f, dst) != nil {
		c.Status(500)
		return
	}
	c.Status(201)
}

func uploadPier(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Status(400)
		return
	}

	_ = os.MkdirAll(pierDir, 0o755)
	dst := filepath.Join(pierDir, filepath.Base(file.Filename))
	i, _ := strconv.Atoi(c.PostForm("dzchunkindex"))
	cnt, _ := strconv.Atoi(c.PostForm("dztotalchunkcount"))
	size, _ := strconv.Atoi(c.PostForm("dztotalfilesize"))
	off, _ := strconv.ParseInt(c.PostForm("dzchunkbyteoffset"), 10, 64)

	w, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		c.Status(500)
		return
	}
	defer w.Close()

	w.Seek(off, io.SeekStart)

	src, _ := file.Open()
	io.Copy(w, src)
	src.Close()
	lastChunk := (i+1 == cnt) || cnt == 0
	if lastChunk {
		fi, _ := w.Stat()
		if cnt == 0 || int(fi.Size()) == size {
			if err := extractAndClean(dst); err != nil {
				log.Println("extract error:", err)
				c.Status(500)
				return
			}
		}
	}

	c.Status(200)
}

func extractAndClean(archive string) error {
	var cmd *exec.Cmd
	switch {
	case strings.HasSuffix(archive, ".zip"):
		cmd = exec.Command("unzip", "-q", archive, "-d", pierDir)
	case strings.HasSuffix(archive, ".tar.gz"):
		cmd = exec.Command("tar", "xzf", archive, "-C", pierDir)
	case strings.HasSuffix(archive, ".tgz"):
		cmd = exec.Command("tar", "xzf", archive, "-C", pierDir)
	default:
		return fmt.Errorf("unsupported archive type: %s", filepath.Base(archive))
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extract failed for %s: %w", archive, err)
	}
	base := filepath.Base(archive)
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".tar.gz"), ".zip")
	name = strings.TrimSuffix(name, ".tgz")
	marker := filepath.Join(pierDir, name, ".extracted")
	if err := os.WriteFile(marker, []byte{}, 0o644); err != nil {
		return err
	}
	if err := os.Remove(archive); err != nil {
		return fmt.Errorf("could not remove archive %s: %w", archive, err)
	}
	return nil
}

func getCode() *json.RawMessage {
	payload := `{"source":{"dojo":"+code"},"sink":{"stdout":null}}`
	resp, err := http.Post(loopback, "application/json", strings.NewReader(payload))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var j json.RawMessage
	json.NewDecoder(resp.Body).Decode(&j)
	return &j
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envInt(k string, d int) int {
	v, _ := strconv.Atoi(env(k, ""))
	if v == 0 {
		return d
	}
	return v
}

func glob(b, p string) []string {
	m, _ := filepath.Glob(filepath.Join(b, p))
	return m
}

func pick(v, d int) int {
	if v != 0 {
		return v
	}
	return d
}
