/*
** Copyright (C) 2026 Key9, Inc <k9.io>
** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
**
** This file is part of the HighVolt JSON analysis engine
**
** This program is free software: you can redistribute it and/or modify
** it under the terms of the GNU Affero General Public License as published by
** the Free Software Foundation, either version 3 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
** GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License
** along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/k9io/highvolt/cmd/highvolt-server/auth"
	"github.com/k9io/highvolt/cmd/highvolt-server/config"
	"github.com/k9io/highvolt/cmd/highvolt-server/db"
	d "github.com/k9io/highvolt/cmd/highvolt-server/debug"
	"github.com/k9io/highvolt/cmd/highvolt-server/models"
	"github.com/k9io/highvolt/cmd/highvolt-server/queue"

	"github.com/k9io/highvolt/internal/droppriv"
	l "github.com/k9io/highvolt/internal/logger"

	"github.com/gin-gonic/gin"
)

var self string = "highvolt-server"

func main() {

	/* We temporary set our logging to "local".  This way if something happens
	   during startup, we can log it.  Once the Highvolt configuration is
	   loaded,  we set it to what the user wants */

	l.Init_Logger("local", "tcp")

	l.Logger(l.INFO, "Firing up HighVolt Server.")

	/* Signal Handler */

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGHUP)

	/* Load the environment (to connect to Redis) */

	config.Load_Env() /* Get environment.  This is needed to login to Redis */

	/* Load Highvolt server config from JSONAir.  This is where the bulk of
	   configuration data is */

	bearerToken := auth.PAT_Auth()

	configJSON, bearerToken := config.GetConfigJSON(bearerToken)
	models.C.JSON = configJSON

	config.Load_Config(configJSON)

	/* Start the configuration monitor */

	go config.Monitor_Config(bearerToken, configJSON)

	go SigHandler(signalChannel, bearerToken)

	/* Reset logging to what the user wants! */

	l.Init_Logger(models.C.Syslog.Host, models.C.Syslog.Proto)

	/* Setup debugging as per the users request */

	err, debug_level := d.GetDebugLevel(bearerToken)

	if err != nil {

		l.Logger(l.ERROR, "%v", err)
		os.Exit(1)

	}

	l.Logger(l.INFO, "Debug level: %s", debug_level)

	/* Connect to Opensearch data queries/data storage */

	db.Init_Opensearch()

	/* Setup "worker" group */

	queue.Init_Queue()

	/* Start HAProxy TCP queue service */

	if models.C.HA_Proxy.Enabled == true {

		go Highvolt_HAProxy_Agent()

	}

	/* Start Queue "goroutine" to monitor new inbound data to analyze */

	go queue.Monitor_Queue()

	/* Start the "goroutine" to monitor configurations and debug updates */

	go Monitor_Reload(bearerToken)

	/* Start HTTP routing */

	router := gin.Default()

	if models.C.HTTP.Mode == "production" {

		gin.SetMode("release")
		gin.DefaultWriter = io.Discard

	} else {

		gin.SetMode(models.C.HTTP.Mode)
		router.Use(HTTP_Logger())

	}

	router.SetTrustedProxies(nil)

	/* Set panic handler */

	router.Use(gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err any) {
		l.Logger(l.ERROR, "Panic recovered: %v\n%s", err, debug.Stack())
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}))

	/* Set some basic security stuff */

	router.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")

		if models.C.HTTP.TLS {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()

	})

	/*
	        if Env.HTTPMode != "production" && Env.HTTPMode != "release" {
	                router.Use(httpLogger())
	} */

	router.Use(func(c *gin.Context) {
		models.ConfigMu.RLock()
		limit := models.C.Core.Max_Body_Size
		models.ConfigMu.RUnlock()
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	})

	router.POST("/api/v1/highvolt/auth/token", rateLimitMiddleware(), authToken)

	configGroup := router.Group("/api/v1/highvolt")

	configGroup.Use(jwtMiddleware())
	{

		configGroup.POST("/submit", Submit)
		configGroup.POST("/query", Query)

	}

	/* Set some sane HTTP defaults */

	server := &http.Server{
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	StartRateLimiterCleanup(ctx)

	if models.C.HTTP.TLS {

		cert, err := tls.LoadX509KeyPair(models.C.HTTP.Cert, models.C.HTTP.Key)

		if err != nil {

			l.Logger(l.ERROR, "Failed to load certificates: %v", err)
			os.Exit(1)

		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		rawListener, err := net.Listen("tcp", models.C.HTTP.Listen)

		if err != nil {

			l.Logger(l.ERROR, "Failed to bind to port '%s': %v", models.C.HTTP.Listen, err)
			os.Exit(1)

		}

		tlsListener := tls.NewListener(rawListener, tlsConfig)

		droppriv.DropPrivileges(models.Env.RUNAS)

		l.Logger(l.INFO, "Listening on '%s' for TLS traffic as UID: %d.", models.C.HTTP.Listen, os.Getuid())

		serveErr := make(chan error, 1)
		go func() {
			if err := server.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
				serveErr <- err
			}
			close(serveErr)
		}()

		select {
		case err := <-serveErr:
			l.Logger(l.ERROR, "Server failed: %v", err)
			os.Exit(1)
		case <-ctx.Done():
			l.Logger(l.INFO, "Shutdown signal received, draining connections...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				l.Logger(l.ERROR, "Server shutdown error: %v", err)
			}
			l.Logger(l.INFO, "Server stopped cleanly.")
		}

	} else {

		ln, err := net.Listen("tcp", models.C.HTTP.Listen)

		if err != nil {

			l.Logger(l.ERROR, "Failed to bind to port '%s': %v", models.C.HTTP.Listen, err)
			os.Exit(1)

		}

		droppriv.DropPrivileges(models.Env.RUNAS)

		l.Logger(l.INFO, "Listening on '%s' for traffic as UID: %d.", models.C.HTTP.Listen, os.Getuid())

		serveErr := make(chan error, 1)

		go func() {
			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				serveErr <- err
			}
			close(serveErr)
		}()

		select {
		case err := <-serveErr:
			l.Logger(l.ERROR, "Server failed: %v", err)
			os.Exit(1)
		case <-ctx.Done():
			l.Logger(l.INFO, "Shutdown signal received, draining connections...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				l.Logger(l.ERROR, "Server shutdown error: %v", err)
			}
			l.Logger(l.INFO, "Server stopped cleanly.")
		}

	}

}
