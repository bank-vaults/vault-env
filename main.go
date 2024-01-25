// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"emperror.dev/errors"
	"github.com/bank-vaults/internal/injector"
	"github.com/bank-vaults/vault-sdk/vault"
	vaultapi "github.com/hashicorp/vault/api"
	slogmulti "github.com/samber/slog-multi"
	slogsyslog "github.com/samber/slog-syslog"
	"github.com/spf13/cast"
)

// The special value for VAULT_ENV which marks that the login token needs to be passed through to the application
// which was acquired during the new Vault client creation
const vaultLogin = "vault:login"

type sanitizedEnviron struct {
	env   []string
	login bool
}

type envType struct {
	login bool
}

var sanitizeEnvmap = map[string]envType{
	"VAULT_TOKEN":                  {login: true},
	"VAULT_ADDR":                   {login: true},
	"VAULT_AGENT_ADDR":             {login: true},
	"VAULT_CACERT":                 {login: true},
	"VAULT_CAPATH":                 {login: true},
	"VAULT_CLIENT_CERT":            {login: true},
	"VAULT_CLIENT_KEY":             {login: true},
	"VAULT_CLIENT_TIMEOUT":         {login: true},
	"VAULT_SRV_LOOKUP":             {login: true},
	"VAULT_SKIP_VERIFY":            {login: true},
	"VAULT_NAMESPACE":              {login: true},
	"VAULT_TLS_SERVER_NAME":        {login: true},
	"VAULT_WRAP_TTL":               {login: true},
	"VAULT_MFA":                    {login: true},
	"VAULT_MAX_RETRIES":            {login: true},
	"VAULT_CLUSTER_ADDR":           {login: false},
	"VAULT_REDIRECT_ADDR":          {login: false},
	"VAULT_CLI_NO_COLOR":           {login: false},
	"VAULT_RATE_LIMIT":             {login: false},
	"VAULT_ROLE":                   {login: false},
	"VAULT_PATH":                   {login: false},
	"VAULT_AUTH_METHOD":            {login: false},
	"VAULT_TRANSIT_KEY_ID":         {login: false},
	"VAULT_TRANSIT_PATH":           {login: false},
	"VAULT_TRANSIT_BATCH_SIZE":     {login: false},
	"VAULT_IGNORE_MISSING_SECRETS": {login: false},
	"VAULT_ENV_PASSTHROUGH":        {login: false},
	"VAULT_JSON_LOG":               {login: false},
	"VAULT_LOG_LEVEL":              {login: false},
	"VAULT_REVOKE_TOKEN":           {login: false},
	"VAULT_ENV_DAEMON":             {login: false},
	"VAULT_ENV_FROM_PATH":          {login: false},
	"VAULT_ENV_DELAY":              {login: false},
}

// Appends variable an entry (name=value) into the environ list.
// VAULT_* variables are not populated into this list if this is not a login scenario.
func (e *sanitizedEnviron) append(name string, value string) {
	if envType, ok := sanitizeEnvmap[name]; !ok || (e.login && envType.login) {
		e.env = append(e.env, fmt.Sprintf("%s=%s", name, value))
	}
}

type daemonSecretRenewer struct {
	client *vault.Client
	sigs   chan os.Signal
	logger *slog.Logger
}

func (r daemonSecretRenewer) Renew(path string, secret *vaultapi.Secret) error {
	watcherInput := vaultapi.LifetimeWatcherInput{Secret: secret}
	watcher, err := r.client.RawClient().NewLifetimeWatcher(&watcherInput)
	if err != nil {
		return errors.Wrap(err, "failed to create secret watcher")
	}

	go watcher.Start()

	go func() {
		defer watcher.Stop()
		for {
			select {
			case renewOutput := <-watcher.RenewCh():
				r.logger.Info("secret renewed", slog.String("path", path), slog.Duration("lease-duration", time.Duration(renewOutput.Secret.LeaseDuration)*time.Second))
			case doneError := <-watcher.DoneCh():
				if !secret.Renewable {
					leaseDuration := time.Duration(secret.LeaseDuration) * time.Second
					time.Sleep(leaseDuration)

					r.logger.Info("secret lease has expired", slog.String("path", path), slog.Duration("lease-duration", leaseDuration))
				}

				r.logger.Info("secret renewal has stopped, sending SIGTERM to process", slog.String("path", path), slog.Any("done-error", doneError))

				r.sigs <- syscall.SIGTERM

				timeout := <-time.After(10 * time.Second)
				r.logger.Info("killing process due to SIGTERM timeout", slog.Time("timeout", timeout))
				r.sigs <- syscall.SIGKILL

				return
			}
		}
	}()

	return nil
}

func main() {
	var logger *slog.Logger
	{
		var level slog.Level

		err := level.UnmarshalText([]byte(os.Getenv("VAULT_LOG_LEVEL")))
		if err != nil { // Silently fall back to info level
			level = slog.LevelInfo
		}

		levelFilter := func(levels ...slog.Level) func(ctx context.Context, r slog.Record) bool {
			return func(ctx context.Context, r slog.Record) bool {
				return slices.Contains(levels, r.Level)
			}
		}

		router := slogmulti.Router()

		if cast.ToBool(os.Getenv("VAULT_JSON_LOG")) {
			// Send logs with level higher than warning to stderr
			router = router.Add(
				slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}),
				levelFilter(slog.LevelWarn, slog.LevelError),
			)

			// Send info and debug logs to stdout
			router = router.Add(
				slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
				levelFilter(slog.LevelDebug, slog.LevelInfo),
			)
		} else {
			// Send logs with level higher than warning to stderr
			router = router.Add(
				slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}),
				levelFilter(slog.LevelWarn, slog.LevelError),
			)

			// Send info and debug logs to stdout
			router = router.Add(
				slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
				levelFilter(slog.LevelDebug, slog.LevelInfo),
			)
		}

		if logServerAddr := os.Getenv("VAULT_ENV_LOG_SERVER"); logServerAddr != "" {
			writer, err := net.Dial("udp", logServerAddr)

			// We silently ignore syslog connection errors for the lack of a better solution
			if err == nil {
				router = router.Add(slogsyslog.Option{Level: slog.LevelInfo, Writer: writer}.NewSyslogHandler())
			}
		}

		// TODO: add level filter handler
		logger = slog.New(router.Handler())
		logger = logger.With(slog.String("app", "vault-env"))

		slog.SetDefault(logger)
	}

	if len(os.Args) == 1 {
		logger.Error("no command is given, vault-env can't determine the entrypoint (command), please specify it explicitly or let the webhook query it (see documentation)")

		os.Exit(1)
	}

	daemonMode := cast.ToBool(os.Getenv("VAULT_ENV_DAEMON"))
	delayExec := cast.ToDuration(os.Getenv("VAULT_ENV_DELAY"))
	sigs := make(chan os.Signal, 1)

	entrypointCmd := os.Args[1:]

	binary, err := exec.LookPath(entrypointCmd[0])
	if err != nil {
		logger.Error("binary not found", slog.String("binary", entrypointCmd[0]))

		os.Exit(1)
	}

	// Used both for reading secrets and transit encryption
	ignoreMissingSecrets := cast.ToBool(os.Getenv("VAULT_IGNORE_MISSING_SECRETS"))

	clientOptions := []vault.ClientOption{vault.ClientLogger(clientLogger{logger})}
	// The login procedure takes the token from a file (if using Vault Agent)
	// or requests one for itself (Kubernetes Auth, or GCP, etc...),
	// so if we got a VAULT_TOKEN for the special value with "vault:login"
	originalVaultTokenEnvVar := os.Getenv("VAULT_TOKEN")
	isLogin := originalVaultTokenEnvVar == vaultLogin
	if tokenFile := os.Getenv("VAULT_TOKEN_FILE"); tokenFile != "" {
		// load token from vault-agent .vault-token or injected webhook
		if b, err := os.ReadFile(tokenFile); err == nil {
			originalVaultTokenEnvVar = string(b)
		} else {
			logger.Error("could not read vault token file", slog.String("file", tokenFile))

			os.Exit(1)
		}
		clientOptions = append(clientOptions, vault.ClientToken(originalVaultTokenEnvVar))
	} else {
		if isLogin {
			_ = os.Unsetenv("VAULT_TOKEN")
		}
		// use role/path based authentication
		clientOptions = append(clientOptions,
			vault.ClientRole(os.Getenv("VAULT_ROLE")),
			vault.ClientAuthPath(os.Getenv("VAULT_PATH")),
			vault.ClientAuthMethod(os.Getenv("VAULT_AUTH_METHOD")),
		)
	}

	client, err := vault.NewClientWithOptions(clientOptions...)
	if err != nil {
		logger.Error(fmt.Errorf("failed to create vault client: %w", err).Error())

		os.Exit(1)
	}

	passthroughEnvVars := strings.Split(os.Getenv("VAULT_ENV_PASSTHROUGH"), ",")

	if isLogin {
		_ = os.Setenv("VAULT_TOKEN", vaultLogin)
		passthroughEnvVars = append(passthroughEnvVars, "VAULT_TOKEN")
	}

	// do not sanitize env vars specified in VAULT_ENV_PASSTHROUGH
	for _, envVar := range passthroughEnvVars {
		if trimmed := strings.TrimSpace(envVar); trimmed != "" {
			delete(sanitizeEnvmap, trimmed)
		}
	}

	// initial and sanitized environs
	environ := make(map[string]string, len(os.Environ()))
	sanitized := sanitizedEnviron{login: isLogin}

	config := injector.Config{
		TransitKeyID:         os.Getenv("VAULT_TRANSIT_KEY_ID"),
		TransitPath:          os.Getenv("VAULT_TRANSIT_PATH"),
		TransitBatchSize:     cast.ToInt(os.Getenv("VAULT_TRANSIT_BATCH_SIZE")),
		DaemonMode:           daemonMode,
		IgnoreMissingSecrets: ignoreMissingSecrets,
	}

	var secretRenewer injector.SecretRenewer

	if daemonMode {
		secretRenewer = daemonSecretRenewer{client: client, sigs: sigs, logger: logger}
	}

	secretInjector := injector.NewSecretInjector(config, client, secretRenewer, logger)

	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		environ[name] = value
	}

	inject := func(key, value string) {
		sanitized.append(key, value)
	}

	err = secretInjector.InjectSecretsFromVault(environ, inject)
	if err != nil {
		logger.Error(fmt.Errorf("failed to inject secrets from vault: %w", err).Error())

		os.Exit(1)
	}

	if paths := os.Getenv("VAULT_ENV_FROM_PATH"); paths != "" {
		err = secretInjector.InjectSecretsFromVaultPath(paths, inject)
	}
	if err != nil {
		logger.Error(fmt.Errorf("failed to inject secrets from vault path: %w", err).Error())

		os.Exit(1)
	}

	if cast.ToBool(os.Getenv("VAULT_REVOKE_TOKEN")) {
		// ref: https://www.vaultproject.io/api/auth/token/index.html#revoke-a-token-self-
		err = client.RawClient().Auth().Token().RevokeSelf(client.RawClient().Token())
		if err != nil {
			// Do not exit on error, token revoking can be denied by policy
			logger.Warn("failed to revoke token")
		}

		client.Close()
	}

	if delayExec > 0 {
		logger.Info(fmt.Sprintf("sleeping for %s...", delayExec))
		time.Sleep(delayExec)
	}

	logger.Info("spawning process", slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

	if daemonMode {
		logger.Info("in daemon mode...")
		cmd := exec.Command(binary, entrypointCmd[1:]...)
		cmd.Env = append(os.Environ(), sanitized.env...)
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		signal.Notify(sigs)

		err = cmd.Start()
		if err != nil {
			logger.Error(fmt.Errorf("failed to start process: %w", err).Error(), slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

			os.Exit(1)
		}

		go func() {
			for sig := range sigs {
				// We don't want to signal a non-running process.
				if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
					break
				}

				err := cmd.Process.Signal(sig)
				if err != nil {
					logger.Warn(fmt.Errorf("failed to signal process: %w", err).Error(), slog.String("signal", sig.String()))
				} else if sig == syscall.SIGURG {
					logger.Debug("received signal", slog.String("signal", sig.String()))
				} else {
					logger.Info("received signal", slog.String("signal", sig.String()))
				}
			}
		}()

		err = cmd.Wait()

		close(sigs)

		if err != nil {
			exitCode := -1
			// try to get the original exit code if possible
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				exitCode = exitError.ExitCode()
			}

			logger.Error(fmt.Errorf("failed to exec process: %w", err).Error(), slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

			os.Exit(exitCode)
		}

		os.Exit(cmd.ProcessState.ExitCode())
	} else { //nolint:revive
		err = syscall.Exec(binary, entrypointCmd, sanitized.env)
		if err != nil {
			logger.Error(fmt.Errorf("failed to exec process: %w", err).Error(), slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

			os.Exit(1)
		}
	}
}
