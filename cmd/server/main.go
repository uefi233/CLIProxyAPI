package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type OpenAIRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func Handler(w http.ResponseWriter, r *http.Request) {

	enableCORS(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path

	switch {

	case path == "/":
		rootHandler(w, r)

	case strings.HasPrefix(path, "/v1/chat/completions"):
		openAIHandler(w, r)

	case strings.HasPrefix(path, "/v1/messages"):
		claudeHandler(w, r)

	case strings.HasPrefix(path, "/v1beta/models"):
		geminiHandler(w, r)

	default:
		notFound(w)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {

	resp := map[string]any{
		"status":  "ok",
		"service": "CLIProxyAPI",
		"runtime": "vercel",
		"time":    time.Now().Unix(),
		"port":    os.Getenv("PORT"),
	}

	writeJSON(w, 200, resp)
}

func openAIHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]any{
			"error": "method not allowed",
		})
		return
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		writeJSON(w, 400, map[string]any{
			"error": err.Error(),
		})
		return
	}

	var req OpenAIRequest

	_ = json.Unmarshal(body, &req)

	userContent := "Hello from CLIProxyAPI"

	if len(req.Messages) > 0 {
		userContent = req.Messages[len(req.Messages)-1].Content
	}

	var resp OpenAIResponse

	resp.ID = "chatcmpl-vercel"
	resp.Object = "chat.completion"
	resp.Created = time.Now().Unix()
	resp.Model = req.Model

	choice := struct {
		Index int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	}{
		Index: 0,
		FinishReason: "stop",
	}

	choice.Message.Role = "assistant"
	choice.Message.Content = "Echo: " + userContent

	resp.Choices = append(resp.Choices, choice)

	writeJSON(w, 200, resp)
}

func claudeHandler(w http.ResponseWriter, r *http.Request) {

	writeJSON(w, 200, map[string]any{
		"id":   "msg_vercel",
		"type": "message",
		"role": "assistant",
		"content": []map[string]any{
			{
				"type": "text",
				"text": "Hello Claude API",
			},
		},
	})
}

func geminiHandler(w http.ResponseWriter, r *http.Request) {

	writeJSON(w, 200, map[string]any{
		"models": []map[string]any{
			{
				"name": "gemini-2.5-pro",
			},
			{
				"name": "gemini-2.5-flash",
			},
		},
	})
}

func notFound(w http.ResponseWriter) {

	writeJSON(w, 404, map[string]any{
		"error": "not found",
	})
}

func enableCORS(w *http.ResponseWriter) {

	(*w).Header().Set(
		"Access-Control-Allow-Origin",
		"*",
	)

	(*w).Header().Set(
		"Access-Control-Allow-Headers",
		"*",
	)

	(*w).Header().Set(
		"Access-Control-Allow-Methods",
		"GET,POST,PUT,DELETE,OPTIONS",
	)

	(*w).Header().Set(
		"Content-Type",
		"application/json",
	)
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {

	w.WriteHeader(status)

	encoder := json.NewEncoder(w)

	encoder.SetIndent("", "  ")

	_ = encoder.Encode(data)
}func main() {
	fmt.Printf(
		"CLIProxyAPI Version: %s, Commit: %s, BuiltAt: %s\n",
		buildinfo.Version,
		buildinfo.Commit,
		buildinfo.BuildDate,
	)

	// 加载 .env
	loadEnv()

	// 获取 PORT
	port := getPort()

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	// 创建 HTTP Router
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(`{
			"status":"ok",
			"service":"CLIProxyAPI",
			"version":"` + buildinfo.Version + `"
		}`))
	})

	// OpenAI Compatible API
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		handleChatCompletions(cfg, w, r)
	})

	// Gemini Compatible API
	mux.HandleFunc("/v1beta/models", func(w http.ResponseWriter, r *http.Request) {
		handleGeminiModels(cfg, w, r)
	})

	// Claude Compatible API
	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		handleClaudeMessages(cfg, w, r)
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       300 * time.Second,
		WriteTimeout:      300 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Infof("server started on port %s", port)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}

func loadEnv() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	envPath := filepath.Join(wd, ".env")

	if err := godotenv.Load(envPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Warnf("failed to load .env: %v", err)
		}
	}
}

func getPort() string {
	port := strings.TrimSpace(os.Getenv("PORT"))

	if port == "" {
		port = "3000"
	}

	// 校验端口是否合法
	if _, err := strconv.Atoi(port); err != nil {
		log.Warn("invalid PORT env, fallback to 3000")
		port = "3000"
	}

	return port
}

func loadConfig() (*config.Config, error) {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		configPath = "./config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set(
			"Access-Control-Allow-Origin",
			"*",
		)

		w.Header().Set(
			"Access-Control-Allow-Headers",
			"*",
		)

		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET,POST,PUT,DELETE,OPTIONS",
		)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleChatCompletions(
	cfg *config.Config,
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	_, _ = w.Write([]byte(`{
		"id":"chatcmpl-vercel",
		"object":"chat.completion",
		"created":1234567890,
		"model":"gpt-4o",
		"choices":[
			{
				"index":0,
				"message":{
					"role":"assistant",
					"content":"Hello from Vercel CLIProxyAPI"
				},
				"finish_reason":"stop"
			}
		]
	}`))
}

func handleGeminiModels(
	cfg *config.Config,
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	_, _ = w.Write([]byte(`{
		"models":[
			{
				"name":"gemini-2.5-pro"
			}
		]
	}`))
}

func handleClaudeMessages(
	cfg *config.Config,
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	_, _ = w.Write([]byte(`{
		"id":"msg_vercel",
		"type":"message",
		"role":"assistant",
		"content":[
			{
				"type":"text",
				"text":"Hello Claude API"
			}
		]
	}`))
}

// Graceful Shutdown（可选）
func shutdownServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancel()

	_ = server.Shutdown(ctx)
}		}
		if parsed == nil {
			parsed = &config.Config{}
		}
		parsed.Home = homeCfg
		parsed.Port = 8317 // Default to 8317 for home mode, can be overridden by home config
		parsed.UsageStatisticsEnabled = true
		cfg = parsed

		// Keep a non-empty config path for downstream components (log paths, management assets, etc),
		// but do not require the file to exist when loading config from home.
		if strings.TrimSpace(configPath) != "" {
			configFilePath = configPath
		} else {
			configFilePath = filepath.Join(wd, "config.yaml")
		}

		// Local stores are intentionally disabled when config is loaded from home.
		usePostgresStore = false
		useObjectStore = false
		useGitStore = false
	} else if usePostgresStore {
		if pgStoreLocalPath == "" {
			pgStoreLocalPath = wd
		}
		pgStoreLocalPath = filepath.Join(pgStoreLocalPath, "pgstore")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		pgStoreInst, err = store.NewPostgresStore(ctx, store.PostgresStoreConfig{
			DSN:      pgStoreDSN,
			Schema:   pgStoreSchema,
			SpoolDir: pgStoreLocalPath,
		})
		cancel()
		if err != nil {
			log.Errorf("failed to initialize postgres token store: %v", err)
			return
		}
		examplePath := filepath.Join(wd, "config.example.yaml")
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		if errBootstrap := pgStoreInst.Bootstrap(ctx, examplePath); errBootstrap != nil {
			cancel()
			log.Errorf("failed to bootstrap postgres-backed config: %v", errBootstrap)
			return
		}
		cancel()
		configFilePath = pgStoreInst.ConfigPath()
		cfg, err = config.LoadConfigOptional(configFilePath, isCloudDeploy)
		if err == nil {
			cfg.AuthDir = pgStoreInst.AuthDir()
			log.Infof("postgres-backed token store enabled, workspace path: %s", pgStoreInst.WorkDir())
		}
	} else if useObjectStore {
		if objectStoreLocalPath == "" {
			if writableBase != "" {
				objectStoreLocalPath = writableBase
			} else {
				objectStoreLocalPath = wd
			}
		}
		objectStoreRoot := filepath.Join(objectStoreLocalPath, "objectstore")
		resolvedEndpoint := strings.TrimSpace(objectStoreEndpoint)
		useSSL := true
		if strings.Contains(resolvedEndpoint, "://") {
			parsed, errParse := url.Parse(resolvedEndpoint)
			if errParse != nil {
				log.Errorf("failed to parse object store endpoint %q: %v", objectStoreEndpoint, errParse)
				return
			}
			switch strings.ToLower(parsed.Scheme) {
			case "http":
				useSSL = false
			case "https":
				useSSL = true
			default:
				log.Errorf("unsupported object store scheme %q (only http and https are allowed)", parsed.Scheme)
				return
			}
			if parsed.Host == "" {
				log.Errorf("object store endpoint %q is missing host information", objectStoreEndpoint)
				return
			}
			resolvedEndpoint = parsed.Host
			if parsed.Path != "" && parsed.Path != "/" {
				resolvedEndpoint = strings.TrimSuffix(parsed.Host+parsed.Path, "/")
			}
		}
		resolvedEndpoint = strings.TrimRight(resolvedEndpoint, "/")
		objCfg := store.ObjectStoreConfig{
			Endpoint:  resolvedEndpoint,
			Bucket:    objectStoreBucket,
			AccessKey: objectStoreAccess,
			SecretKey: objectStoreSecret,
			LocalRoot: objectStoreRoot,
			UseSSL:    useSSL,
			PathStyle: true,
		}
		objectStoreInst, err = store.NewObjectTokenStore(objCfg)
		if err != nil {
			log.Errorf("failed to initialize object token store: %v", err)
			return
		}
		examplePath := filepath.Join(wd, "config.example.yaml")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if errBootstrap := objectStoreInst.Bootstrap(ctx, examplePath); errBootstrap != nil {
			cancel()
			log.Errorf("failed to bootstrap object-backed config: %v", errBootstrap)
			return
		}
		cancel()
		configFilePath = objectStoreInst.ConfigPath()
		cfg, err = config.LoadConfigOptional(configFilePath, isCloudDeploy)
		if err == nil {
			if cfg == nil {
				cfg = &config.Config{}
			}
			cfg.AuthDir = objectStoreInst.AuthDir()
			log.Infof("object-backed token store enabled, bucket: %s", objectStoreBucket)
		}
	} else if useGitStore {
		if gitStoreLocalPath == "" {
			if writableBase != "" {
				gitStoreLocalPath = writableBase
			} else {
				gitStoreLocalPath = wd
			}
		}
		gitStoreRoot = filepath.Join(gitStoreLocalPath, "gitstore")
		authDir := filepath.Join(gitStoreRoot, "auths")
		gitStoreInst = store.NewGitTokenStore(gitStoreRemoteURL, gitStoreUser, gitStorePassword, gitStoreBranch)
		gitStoreInst.SetBaseDir(authDir)
		if errRepo := gitStoreInst.EnsureRepository(); errRepo != nil {
			log.Errorf("failed to prepare git token store: %v", errRepo)
			return
		}
		configFilePath = gitStoreInst.ConfigPath()
		if configFilePath == "" {
			configFilePath = filepath.Join(gitStoreRoot, "config", "config.yaml")
		}
		if _, statErr := os.Stat(configFilePath); errors.Is(statErr, fs.ErrNotExist) {
			examplePath := filepath.Join(wd, "config.example.yaml")
			if _, errExample := os.Stat(examplePath); errExample != nil {
				log.Errorf("failed to find template config file: %v", errExample)
				return
			}
			if errCopy := misc.CopyConfigTemplate(examplePath, configFilePath); errCopy != nil {
				log.Errorf("failed to bootstrap git-backed config: %v", errCopy)
				return
			}
			if errCommit := gitStoreInst.PersistConfig(context.Background()); errCommit != nil {
				log.Errorf("failed to commit initial git-backed config: %v", errCommit)
				return
			}
			log.Infof("git-backed config initialized from template: %s", configFilePath)
		} else if statErr != nil {
			log.Errorf("failed to inspect git-backed config: %v", statErr)
			return
		}
		cfg, err = config.LoadConfigOptional(configFilePath, isCloudDeploy)
		if err == nil {
			cfg.AuthDir = gitStoreInst.AuthDir()
			log.Infof("git-backed token store enabled, repository path: %s", gitStoreRoot)
		}
	} else if configPath != "" {
		configFilePath = configPath
		cfg, err = config.LoadConfigOptional(configPath, isCloudDeploy)
	} else {
		wd, err = os.Getwd()
		if err != nil {
			log.Errorf("failed to get working directory: %v", err)
			return
		}
		configFilePath = filepath.Join(wd, "config.yaml")
		cfg, err = config.LoadConfigOptional(configFilePath, isCloudDeploy)
	}
	if err != nil {
		log.Errorf("failed to load config: %v", err)
		return
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	// In cloud deploy mode, check if we have a valid configuration
	var configFileExists bool
	if isCloudDeploy {
		if configLoadedFromHome && cfg != nil {
			configFileExists = cfg.Port != 0
		} else {
			if info, errStat := os.Stat(configFilePath); errStat != nil {
				// Don't mislead: API server will not start until configuration is provided.
				log.Info("Cloud deploy mode: No configuration file detected; standing by for configuration")
				configFileExists = false
			} else if info.IsDir() {
				log.Info("Cloud deploy mode: Config path is a directory; standing by for configuration")
				configFileExists = false
			} else if cfg.Port == 0 {
				// LoadConfigOptional returns empty config when file is empty or invalid.
				// Config file exists but is empty or invalid; treat as missing config
				log.Info("Cloud deploy mode: Configuration file is empty or invalid; standing by for valid configuration")
				configFileExists = false
			} else {
				log.Info("Cloud deploy mode: Configuration file detected; starting service")
				configFileExists = true
			}
		}
	}
	redisqueue.SetUsageStatisticsEnabled(cfg.UsageStatisticsEnabled)
	redisqueue.SetRetentionSeconds(cfg.RedisUsageQueueRetentionSeconds)
	coreauth.SetQuotaCooldownDisabled(cfg.DisableCooling)

	if err = logging.ConfigureLogOutput(cfg); err != nil {
		log.Errorf("failed to configure log output: %v", err)
		return
	}

	log.Infof("CLIProxyAPI Version: %s, Commit: %s, BuiltAt: %s", buildinfo.Version, buildinfo.Commit, buildinfo.BuildDate)

	// Set the log level based on the configuration.
	util.SetLogLevel(cfg)

	if resolvedAuthDir, errResolveAuthDir := util.ResolveAuthDir(cfg.AuthDir); errResolveAuthDir != nil {
		log.Errorf("failed to resolve auth directory: %v", errResolveAuthDir)
		return
	} else {
		cfg.AuthDir = resolvedAuthDir
	}
	managementasset.SetCurrentConfig(cfg)

	// Create login options to be used in authentication flows.
	options := &cmd.LoginOptions{
		NoBrowser:    noBrowser,
		CallbackPort: oauthCallbackPort,
	}

	// Register the shared token store once so all components use the same persistence backend.
	if usePostgresStore {
		sdkAuth.RegisterTokenStore(pgStoreInst)
	} else if useObjectStore {
		sdkAuth.RegisterTokenStore(objectStoreInst)
	} else if useGitStore {
		sdkAuth.RegisterTokenStore(gitStoreInst)
	} else {
		sdkAuth.RegisterTokenStore(sdkAuth.NewFileTokenStore())
	}

	// Register built-in access providers before constructing services.
	configaccess.Register(&cfg.SDKConfig)

	// Handle different command modes based on the provided flags.

	if vertexImport != "" {
		// Handle Vertex service account import
		cmd.DoVertexImport(cfg, vertexImport, vertexImportPrefix)
	} else if login {
		// Handle Google/Gemini login
		cmd.DoLogin(cfg, projectID, options)
	} else if antigravityLogin {
		// Handle Antigravity login
		cmd.DoAntigravityLogin(cfg, options)
	} else if codexLogin {
		// Handle Codex login
		cmd.DoCodexLogin(cfg, options)
	} else if codexDeviceLogin {
		// Handle Codex device-code login
		cmd.DoCodexDeviceLogin(cfg, options)
	} else if claudeLogin {
		// Handle Claude login
		cmd.DoClaudeLogin(cfg, options)
	} else if kimiLogin {
		cmd.DoKimiLogin(cfg, options)
	} else if xaiLogin {
		cmd.DoXAILogin(cfg, options)
	} else {
		// In cloud deploy mode without config file, just wait for shutdown signals
		if isCloudDeploy && !configFileExists {
			// No config file available, just wait for shutdown
			cmd.WaitForCloudDeploy()
			return
		}
		if localModel && (!tuiMode || standalone) {
			log.Info("Local model mode: using embedded model catalog, remote model updates disabled")
		}
		if tuiMode {
			if standalone {
				// Standalone mode: start an embedded local server and connect TUI client to it.
				managementasset.StartAutoUpdater(context.Background(), configFilePath)
				misc.StartAntigravityVersionUpdater(context.Background())
				if !localModel && !cfg.Home.Enabled {
					registry.StartModelsUpdater(context.Background())
				} else if cfg.Home.Enabled {
					log.Info("Home mode: remote model updates disabled")
				}
				hook := tui.NewLogHook(2000)
				hook.SetFormatter(&logging.LogFormatter{})
				log.AddHook(hook)

				origStdout := os.Stdout
				origStderr := os.Stderr
				origLogOutput := log.StandardLogger().Out
				log.SetOutput(io.Discard)

				devNull, errOpenDevNull := os.Open(os.DevNull)
				if errOpenDevNull == nil {
					os.Stdout = devNull
					os.Stderr = devNull
				}

				restoreIO := func() {
					os.Stdout = origStdout
					os.Stderr = origStderr
					log.SetOutput(origLogOutput)
					if devNull != nil {
						_ = devNull.Close()
					}
				}

				localMgmtPassword := fmt.Sprintf("tui-%d-%d", os.Getpid(), time.Now().UnixNano())
				if password == "" {
					password = localMgmtPassword
				}

				cancel, done := cmd.StartServiceBackground(cfg, configFilePath, password)

				client := tui.NewClient(cfg.Port, password)
				ready := false
				backoff := 100 * time.Millisecond
				for i := 0; i < 30; i++ {
					if _, errGetConfig := client.GetConfig(); errGetConfig == nil {
						ready = true
						break
					}
					time.Sleep(backoff)
					if backoff < time.Second {
						backoff = time.Duration(float64(backoff) * 1.5)
					}
				}

				if !ready {
					restoreIO()
					cancel()
					<-done
					fmt.Fprintf(os.Stderr, "TUI error: embedded server is not ready\n")
					return
				}

				if errRun := tui.Run(cfg.Port, password, hook, origStdout); errRun != nil {
					restoreIO()
					fmt.Fprintf(os.Stderr, "TUI error: %v\n", errRun)
				} else {
					restoreIO()
				}

				cancel()
				<-done
			} else {
				// Default TUI mode: pure management client.
				// The proxy server must already be running.
				if errRun := tui.Run(cfg.Port, password, nil, os.Stdout); errRun != nil {
					fmt.Fprintf(os.Stderr, "TUI error: %v\n", errRun)
				}
			}
		} else {
			// Start the main proxy service
			managementasset.StartAutoUpdater(context.Background(), configFilePath)
			misc.StartAntigravityVersionUpdater(context.Background())
			if !localModel && !cfg.Home.Enabled {
				registry.StartModelsUpdater(context.Background())
			} else if cfg.Home.Enabled {
				log.Info("Home mode: remote model updates disabled")
			}
			cmd.StartService(cfg, configFilePath, password)
		}
	}
}
