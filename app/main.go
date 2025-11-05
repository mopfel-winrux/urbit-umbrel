package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
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
	"path/filepath"
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

const sessKey = "umbrel_user"

var (
	keyDir      = env("KEY_DIR", "/data/keys")
	pierDir     = env("PIER_DIR", "/data/piers")
	appPort     = env("APP_PORT", "8090")
	amesPort    = envInt("AMES_PORT", 34343)
	defaultLoom = envInt("DEFAULT_LOOM", 31)
	loopback    = "http://127.0.0.1:12321"
	appPwd      = env("APP_PASSWORD", "")
	urbit       *runner
)

type runner struct {
	cmd     *exec.Cmd
	args    []string
	buf     bytes.Buffer
	mu      sync.Mutex
	started time.Time
	once    sync.Once
	quit    chan struct{}
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
		authed.POST("/boot", bootExisting)
		authed.POST("/boot-comet", bootComet)
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
		if urbit != nil {
			if up, _, _, _ := urbit.snap(); up {
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

	log.Fatal(r.Run(fmt.Sprintf(":%s", appPort)))
}

func newRunner(args []string) *runner {
	r := &runner{
		args: args,
		quit: make(chan struct{}),
	}
	r.start()
	go func() {
		_ = r.cmd.Wait()
		r.once.Do(func() { close(r.quit) })
	}()
	return r
}

func (r *runner) start() {
	r.mu.Lock()
	r.cmd = exec.Command("/usr/bin/urbit", r.args...)
	stdout, _ := r.cmd.StdoutPipe()
	stderr, _ := r.cmd.StderrPipe()
	if err := r.cmd.Start(); err != nil {
		r.mu.Unlock()
		log.Println("urbit start error:", err)
		return
	}
	r.started = time.Now()
	r.mu.Unlock()

	go r.tail(stdout)
	go r.tail(stderr)
	go func() {
		r.cmd.Wait()
		r.once.Do(func() { close(r.quit) })
	}()
}

func (r *runner) stop() {
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Signal(syscall.SIGTERM)
		go func(p *os.Process) {
			time.Sleep(3 * time.Second)
			p.Kill()
		}(r.cmd.Process)
	}
	r.once.Do(func() { close(r.quit) })
}

func (r *runner) snap() (up bool, upFor time.Duration, tail string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	up = r.cmd != nil && r.cmd.Process != nil &&
		r.cmd.Process.Signal(syscall.Signal(0)) == nil
	if up {
		upFor = time.Since(r.started)
	}
	tail = r.buf.String()
	return up, upFor, tail, nil
}

func (r *runner) tail(rc io.ReadCloser) {
	sc := bufio.NewScanner(rc)
	for sc.Scan() {
		b := sc.Bytes()
		r.mu.Lock()
		r.buf.Write(b)
		r.buf.WriteByte('\n')
		if r.buf.Len() > 4096 {
			r.buf.Next(r.buf.Len() - 4096)
		}
		r.mu.Unlock()
		os.Stdout.Write(append(b, '\n'))
	}
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
	if urbit != nil {
		upDur, _, _, _ := func() (time.Duration, bool, string, error) {
			u, d, t, e := urbit.snap()
			return d, u, t, e
		}()
		running = true
		upFor = upDur.Seconds()
	}
	state := "stopped"
	if running {
		if getCode() == nil {
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
		name := e.Name()
		dir := filepath.Join(pierDir, name)
		if strings.HasPrefix(name, "comet-") {
			piers = append(piers, dir+"/")
			continue
		}
		piers = append(piers, dir+"/")
	}
	c.JSON(200, gin.H{
		"keys":         glob(keyDir, "*.key"),
		"piers":        piers,
		"loomValues":   []int{31, 32, 33},
		"code":         getCode(),
		"urbitRunning": running,
		"uptime":       upFor,
		"state":        state,
	})
}

func logs(c *gin.Context) {
	if urbit == nil {
		c.JSON(200, gin.H{"running": false})
		return
	}
	up, dur, tail, err := urbit.snap()
	c.JSON(200, gin.H{
		"running": up,
		"uptime":  int(dur.Seconds()),
		"tail":    tail,
		"error":   fmt.Sprint(err),
	})
}

func stopUrbit(c *gin.Context) {
	if urbit != nil {
		urbit.stop()
		urbit = nil
	}
	c.Status(202)
}

func startUrbit(args []string) { urbit = newRunner(args) }

func bootExisting(c *gin.Context) {
	if urbit != nil {
		urbit.stop()
		urbit = nil
	}
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
		startUrbit([]string{
			"-t",
			"--http-port", "8000",
			"-b", "0.0.0.0",
			"-p", strconv.Itoa(amesPort),
			"--loom", strconv.Itoa(loom),
			"-w", ship,
			"-k", req.Path,
			"-c", pier,
		})
		go func(keyPath string) {
			for i := 0; i < 5; i++ {
				time.Sleep(30 * time.Second)
				err := os.Remove(keyPath)
				if err == nil {
					break
				}
			}
		}(req.Path)

	case strings.HasPrefix(req.Path, "/data/piers/"):
		pier := filepath.Join(pierDir, filepath.Base(req.Path))
		startUrbit([]string{
			"-t", "-b", "0.0.0.0",
			"--http-port", "8000",
			"-p", strconv.Itoa(amesPort),
			"--loom", strconv.Itoa(loom),
			pier,
		})

	default:
		c.Status(http.StatusBadRequest)
		return
	}

	c.Status(202)
}

func bootComet(c *gin.Context) {
	if urbit != nil {
		urbit.stop()
		urbit = nil
	}
	var req struct{ Loom int }
	_ = c.BindJSON(&req)

	rand.Seed(time.Now().UnixNano())
	name := fmt.Sprintf("comet-%d", rand.Intn(1<<16))
	pier := filepath.Join(pierDir, name)

	startUrbit([]string{
		"-t",
		"-p", strconv.Itoa(amesPort),
		"--http-port", "8000",
		"--loom", strconv.Itoa(pick(req.Loom, defaultLoom)),
		"-c", pier,
	})

	c.Status(202)
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
func glob(b, p string) []string { m, _ := filepath.Glob(filepath.Join(b, p)); return m }
func pick(v, d int) int {
	if v != 0 {
		return v
	}
	return d
}
